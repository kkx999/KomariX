package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kkx999/KomariX/database/accounts"
	"github.com/kkx999/KomariX/database/auditlog"
	"github.com/kkx999/KomariX/internal/config"
	"github.com/kkx999/KomariX/utils"
	"github.com/kkx999/KomariX/web/api"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TwoFa    string `json:"2fa_code"`
}

const sessionCookieMaxAge = 2592000


const (
	maxLoginRequestBytes = 64 << 10
	loginFailureLimit    = 10
	loginBlockDuration   = 15 * time.Minute
	loginFailureWindow   = 30 * time.Minute
)

type loginFailureState struct {
	Failures     int
	LastFailure  time.Time
	BlockedUntil time.Time
}

var loginFailureTracker = struct {
	sync.Mutex
	entries   map[string]loginFailureState
	lastSweep time.Time
}{entries: make(map[string]loginFailureState)}

func sweepLoginFailuresLocked(now time.Time) {
	if !loginFailureTracker.lastSweep.IsZero() && now.Sub(loginFailureTracker.lastSweep) < time.Minute {
		return
	}
	for ip, state := range loginFailureTracker.entries {
		if now.After(state.BlockedUntil) && now.Sub(state.LastFailure) > loginFailureWindow {
			delete(loginFailureTracker.entries, ip)
		}
	}
	loginFailureTracker.lastSweep = now
}

func loginBlocked(ip string) (bool, time.Duration) {
	now := time.Now().UTC()
	loginFailureTracker.Lock()
	defer loginFailureTracker.Unlock()
	sweepLoginFailuresLocked(now)
	state, ok := loginFailureTracker.entries[ip]
	if !ok || state.BlockedUntil.IsZero() || !now.Before(state.BlockedUntil) {
		return false, 0
	}
	return true, time.Until(state.BlockedUntil)
}

func recordLoginFailure(ip string) (attempt int, blocked bool) {
	now := time.Now().UTC()
	loginFailureTracker.Lock()
	defer loginFailureTracker.Unlock()
	sweepLoginFailuresLocked(now)
	state := loginFailureTracker.entries[ip]
	if state.LastFailure.IsZero() || now.Sub(state.LastFailure) > loginFailureWindow || (!state.BlockedUntil.IsZero() && !now.Before(state.BlockedUntil)) {
		state.Failures = 0
		state.BlockedUntil = time.Time{}
	}
	state.Failures++
	state.LastFailure = now
	attempt = state.Failures
	if state.Failures >= loginFailureLimit {
		state.BlockedUntil = now.Add(loginBlockDuration)
		blocked = true
	}
	loginFailureTracker.entries[ip] = state
	return attempt, blocked
}

func clearLoginFailures(ip string) {
	loginFailureTracker.Lock()
	delete(loginFailureTracker.entries, ip)
	loginFailureTracker.Unlock()
}

func safeLoginName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '?'
		}
		return r
	}, value)
	if len(value) > 128 {
		value = value[:128]
	}
	return value
}

func setSessionCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   utils.GetScheme(c) == "https",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func Login(c *gin.Context) {
	DisablePasswordLogin, _ := config.GetAs[bool](config.DisablePasswordLoginKey, false)
	if DisablePasswordLogin {
		api.RespondError(c, http.StatusForbidden, "Password login is disabled")
		return
	}

	ip := c.ClientIP()
	if blocked, remaining := loginBlocked(ip); blocked {
		seconds := int(remaining.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		c.Header("Retry-After", strconv.Itoa(seconds))
		api.RespondError(c, http.StatusTooManyRequests, "Too many failed login attempts. Try again later.")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLoginRequestBytes)
	var data LoginRequest
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(&data); err != nil {
		api.RespondError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	if data.Username == "" || data.Password == "" {
		api.RespondError(c, http.StatusBadRequest, "Invalid request body: Username and password are required")
		return
	}

	uuid, success := accounts.CheckPassword(data.Username, data.Password)
	if !success {
		attempt, newlyBlocked := recordLoginFailure(ip)
		name := safeLoginName(data.Username)
		auditlog.Log(ip, "", fmt.Sprintf("login failed for user %q: invalid credentials (%d/%d)", name, attempt, loginFailureLimit), "security")
		if newlyBlocked {
			auditlog.Log(ip, "", fmt.Sprintf("IP blocked for %s after %d failed login attempts", loginBlockDuration, loginFailureLimit), "security")
		}
		api.RespondError(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	// 2FA
	user, _ := accounts.GetUserByUUID(uuid)
	if user.TwoFactor != "" { // 开启了2FA
		if data.TwoFa == "" {
			attempt, newlyBlocked := recordLoginFailure(ip)
			auditlog.Log(ip, uuid, fmt.Sprintf("login failed: 2FA code required (%d/%d)", attempt, loginFailureLimit), "security")
			if newlyBlocked {
				auditlog.Log(ip, uuid, fmt.Sprintf("IP blocked for %s after %d failed login attempts", loginBlockDuration, loginFailureLimit), "security")
			}
			api.RespondError(c, http.StatusUnauthorized, "2FA code is required")
			return
		}
		if ok, err := accounts.Verify2Fa(uuid, data.TwoFa); err != nil || !ok {
			attempt, newlyBlocked := recordLoginFailure(ip)
			auditlog.Log(ip, uuid, fmt.Sprintf("login failed: invalid 2FA code (%d/%d)", attempt, loginFailureLimit), "security")
			if newlyBlocked {
				auditlog.Log(ip, uuid, fmt.Sprintf("IP blocked for %s after %d failed login attempts", loginBlockDuration, loginFailureLimit), "security")
			}
			api.RespondError(c, http.StatusUnauthorized, "Invalid 2FA code")
			return
		}
	}
	// Create session
	session, err := accounts.CreateSession(uuid, sessionCookieMaxAge, c.Request.UserAgent(), c.ClientIP(), "password")
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "Failed to create session: "+err.Error())
		return
	}
	clearLoginFailures(ip)
	setSessionCookie(c, session, sessionCookieMaxAge)
	auditlog.Log(ip, uuid, "logged in (password)", "login")
	api.RespondSuccess(c, gin.H{"set-cookie": gin.H{"session_token": session}})
}
func Logout(c *gin.Context) {
	session, _ := c.Cookie("session_token")
	accounts.DeleteSession(session)
	setSessionCookie(c, "", -1)
	auditlog.Log(c.ClientIP(), "", "logged out", "logout")
	c.Redirect(302, "/")
}
