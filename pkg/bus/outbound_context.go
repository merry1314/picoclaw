package bus

import "strings"

// NewOutboundContext builds the minimal normalized addressing context required
// to deliver an outbound text message or reply.
func NewOutboundContext(channel, chatID, replyToMessageID string) InboundContext {
	return normalizeInboundContext(InboundContext{
		Channel:          strings.TrimSpace(channel),
		ChatID:           strings.TrimSpace(chatID),
		ReplyToMessageID: strings.TrimSpace(replyToMessageID),
	})
}

// NormalizeOutboundMessage ensures Context is normalized and keeps convenience
// mirrors in sync for runtime consumers.
func NormalizeOutboundMessage(msg OutboundMessage) OutboundMessage {
	msg.Context = normalizeInboundContext(msg.Context)
	msg.Channel = msg.Context.Channel
	msg.ChatID = msg.Context.ChatID
	if msg.Context.ReplyToMessageID == "" {
		msg.Context.ReplyToMessageID = strings.TrimSpace(msg.ReplyToMessageID)
	}
	msg.ReplyToMessageID = msg.Context.ReplyToMessageID
	return msg
}

// NormalizeOutboundMediaMessage ensures media outbound messages also carry a
// normalized context while keeping convenience mirrors in sync.
func NormalizeOutboundMediaMessage(msg OutboundMediaMessage) OutboundMediaMessage {
	msg.Context = normalizeInboundContext(msg.Context)
	msg.Channel = msg.Context.Channel
	msg.ChatID = msg.Context.ChatID
	return msg
}
