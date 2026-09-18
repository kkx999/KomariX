package messageSender

import (
	_ "github.com/kkx999/KomariX/utils/messageSender/bark"
	_ "github.com/kkx999/KomariX/utils/messageSender/email"
	_ "github.com/kkx999/KomariX/utils/messageSender/empty"
	_ "github.com/kkx999/KomariX/utils/messageSender/javascript"
	_ "github.com/kkx999/KomariX/utils/messageSender/serverchan3"
	_ "github.com/kkx999/KomariX/utils/messageSender/serverchanturbo"
	_ "github.com/kkx999/KomariX/utils/messageSender/telegram"
	_ "github.com/kkx999/KomariX/utils/messageSender/webhook"
)

func All() {
}
