package setup

import (
	"time"

	"github.com/TicketsBot/common/permission"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/command"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/command/registry"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/customisation"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/dbclient"
	"github.com/jadevelopmentgrp/Tickets-Worker/i18n"
	"github.com/rxdn/gdl/objects/interaction"
)

type LimitSetupCommand struct{}

func (LimitSetupCommand) Properties() registry.Properties {
	return registry.Properties{
		Name:            "limit",
		Description:     i18n.HelpSetup,
		Type:            interaction.ApplicationCommandTypeChatInput,
		PermissionLevel: permission.Admin,
		Category:        command.Settings,
		Arguments: command.Arguments(
			command.NewRequiredArgument("limit", "The maximum amount of tickets a user can have open simultaneously", interaction.OptionTypeInteger, i18n.SetupLimitInvalid),
		),
		Timeout: time.Second * 3,
	}
}

func (c LimitSetupCommand) GetExecutor() interface{} {
	return c.Execute
}

func (LimitSetupCommand) Execute(ctx registry.CommandContext, limit int) {
	if limit < 1 || limit > 10 {
		ctx.Reply(customisation.Red, i18n.TitleSetup, i18n.SetupLimitInvalid)
		return
	}

	if err := dbclient.Client.TicketLimit.Set(ctx, ctx.GuildId(), uint8(limit)); err != nil {
		ctx.HandleError(err)
		return
	}

	ctx.Reply(customisation.Green, i18n.TitleSetup, i18n.SetupLimitComplete, limit)
}
