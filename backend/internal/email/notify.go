package email

import (
	"nexus-forum-backend/internal/model"
)

func MaybeNotifyForNotification(mailer *Mailer, user *model.User, notif *model.Notification) {
	if mailer == nil || !mailer.Enabled() || user == nil || user.Email == "" || notif == nil {
		return
	}

	allowed := false
	subject := notif.Title
	if subject == "" {
		subject = "Nexus Forum notification"
	}
	body := notif.Body

	switch notif.Type {
	case "reply":
		allowed = user.EmailNotifyReply
	case "mention":
		allowed = user.EmailNotifyMention
	case "follow":
		allowed = user.EmailNotifyFollow
	case "moderation", "content_removed", "user_banned":
		allowed = user.EmailNotifyModeration
	case "report_resolved", "report_rejected":
		allowed = user.EmailNotifyReport
	default:
		return
	}

	if !allowed {
		return
	}
	_ = mailer.SendNotification(user.Email, subject, body)
}
