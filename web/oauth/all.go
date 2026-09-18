package oauth

import (
	_ "github.com/kkx999/KomariX/web/oauth/factory"
	_ "github.com/kkx999/KomariX/web/oauth/generic"
	_ "github.com/kkx999/KomariX/web/oauth/github"
	_ "github.com/kkx999/KomariX/web/oauth/qq"
)

func All() {
	//empty function to ensure all OIDC providers are registered
}
