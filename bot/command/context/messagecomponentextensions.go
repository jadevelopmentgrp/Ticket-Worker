package context

import (
	"context"
	"fmt"

	"github.com/jadevelopmentgrp/Tickets-Worker/bot/button"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/command"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/command/registry"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/customisation"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/utils"
	"github.com/jadevelopmentgrp/Tickets-Worker/i18n"
	"github.com/rxdn/gdl/objects/interaction"
	"github.com/rxdn/gdl/objects/interaction/component"
	"github.com/rxdn/gdl/rest"
	"go.uber.org/atomic"
)

type MessageComponentExtensions struct {
	ctx             registry.CommandContext
	interaction     interaction.InteractionMetadata
	responseChannel chan button.Response
	hasReplied      *atomic.Bool
}

func NewMessageComponentExtensions(
	ctx registry.CommandContext,
	interaction interaction.InteractionMetadata,
	responseChannel chan button.Response,
	hasReplied *atomic.Bool,
) *MessageComponentExtensions {
	return &MessageComponentExtensions{
		ctx:             ctx,
		interaction:     interaction,
		responseChannel: responseChannel,
		hasReplied:      hasReplied,
	}
}

func (e *MessageComponentExtensions) Modal(res button.ResponseModal) {
	e.hasReplied.Store(true)
	e.responseChannel <- res
}

func (e *MessageComponentExtensions) Ack() {
	e.hasReplied.Store(true)
	//e.responseChannel <- button.ResponseAck{}
}

func (e *MessageComponentExtensions) Edit(data command.MessageResponse) {
	hasReplied := e.hasReplied.Swap(true)

	if !hasReplied {
		e.responseChannel <- button.ResponseEdit{
			Data: data,
		}
	} else {
		_, err := rest.EditOriginalInteractionResponse(context.Background(), e.interaction.Token, e.ctx.Worker().RateLimiter, e.ctx.Worker().BotId, data.IntoWebhookEditBody())
		if err != nil {
			fmt.Print(err, e.ctx.ToErrorContext())
		}
	}

	return
}

func (e *MessageComponentExtensions) EditWith(colour customisation.Colour, title, content i18n.MessageId, format ...interface{}) {
	e.Edit(command.MessageResponse{
		Embeds: utils.Slice(utils.BuildEmbed(e.ctx, colour, title, content, nil, format...)),
	})
}

func (e *MessageComponentExtensions) EditWithRaw(colour customisation.Colour, title, content string) {
	e.Edit(command.MessageResponse{
		Embeds: utils.Slice(utils.BuildEmbedRaw(e.ctx.GetColour(colour), title, content, nil)),
	})
}

func (e *MessageComponentExtensions) EditWithComponents(colour customisation.Colour, title, content i18n.MessageId, components []component.Component, format ...interface{}) {
	e.Edit(command.MessageResponse{
		Embeds:     utils.Slice(utils.BuildEmbed(e.ctx, colour, title, content, nil, format...)),
		Components: components,
	})
}

func (e *MessageComponentExtensions) EditWithComponentsRaw(colour customisation.Colour, title, content string, components []component.Component) {
	e.Edit(command.MessageResponse{
		Embeds:     utils.Slice(utils.BuildEmbedRaw(e.ctx.GetColour(colour), title, content, nil)),
		Components: components,
	})
}
