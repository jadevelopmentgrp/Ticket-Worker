package handlers

import (
	"errors"
	"fmt"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/worker/bot/button"
	"github.com/TicketsBot/worker/bot/button/registry"
	"github.com/TicketsBot/worker/bot/button/registry/matcher"
	"github.com/TicketsBot/worker/bot/command/context"
	"github.com/TicketsBot/worker/bot/customisation"
	"github.com/TicketsBot/worker/bot/dbclient"
	"github.com/TicketsBot/worker/bot/logic"
	"github.com/TicketsBot/worker/i18n"
	"github.com/rxdn/gdl/objects/interaction"
	"github.com/rxdn/gdl/objects/interaction/component"
)

type MultiPanelHandler struct{}

func (h *MultiPanelHandler) Matcher() matcher.Matcher {
	return &matcher.SimpleMatcher{
		CustomId: "multipanel",
	}
}

func (h *MultiPanelHandler) Properties() registry.Properties {
	return registry.Properties{
		Flags: registry.SumFlags(registry.GuildAllowed),
	}
}

func (h *MultiPanelHandler) Execute(ctx *context.SelectMenuContext) {
	if len(ctx.InteractionData.Values) == 0 {
		return
	}

	panelCustomId := ctx.InteractionData.Values[0]

	panel, ok, err := dbclient.Client.Panel.GetByCustomId(ctx.GuildId(), panelCustomId)
	if err != nil {
		sentry.Error(err) // TODO: Proper context
		return
	}

	if ok {
		// TODO: Log this
		if panel.GuildId != ctx.GuildId() {
			return
		}

		// blacklist check
		blacklisted, err := dbclient.Client.Blacklist.IsBlacklisted(panel.GuildId, ctx.InteractionUser().Id)
		if err != nil {
			ctx.HandleError(err)
			return
		}

		if blacklisted {
			ctx.Reply(customisation.Red, i18n.TitleBlacklisted, i18n.MessageBlacklisted)
			return
		}

		if panel.FormId == nil {
			_, _ = logic.OpenTicket(ctx, &panel, panel.Title, nil)
		} else {
			form, ok, err := dbclient.Client.Forms.Get(*panel.FormId)
			if err != nil {
				ctx.HandleError(err)
				return
			}

			if !ok {
				ctx.HandleError(errors.New("Form not found"))
				return
			}

			inputs, err := dbclient.Client.FormInput.GetInputs(form.Id)
			if err != nil {
				ctx.HandleError(err)
				return
			}

			if len(inputs) == 0 { // Don't open a blank form
				_, _ = logic.OpenTicket(ctx, &panel, panel.Title, nil)
			} else {
				components := make([]component.Component, len(inputs))
				for i, input := range inputs {
					style := component.TextStyleTypes(input.Style) // wrap

					var maxLength uint32
					if style == component.TextStyleShort {
						maxLength = 255
					} else if style == component.TextStyleParagraph {
						maxLength = 1024 // Max embed field value
					}

					components[i] = component.BuildActionRow(component.BuildInputText(component.InputText{
						Style:       component.TextStyleTypes(input.Style),
						CustomId:    input.CustomId,
						Label:       input.Label,
						Placeholder: input.Placeholder,
						MinLength:   nil,
						MaxLength:   &maxLength,
					}))
				}

				modal := button.ResponseModal{
					Data: interaction.ModalResponseData{
						CustomId:   fmt.Sprintf("form_%s", panel.CustomId),
						Title:      form.Title,
						Components: components,
					},
				}

				ctx.Modal(modal)
			}
		}
	}
}