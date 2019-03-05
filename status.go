package botmaid

func (bm *BotMaid) status(u *Update, b *Bot) bool {
	args := SplitCommand(u.Message.Text)
	if b.IsCommand(u, "status") && len(args) == 1 {
		b.Reply(u, "√")
		return true
	}
	return false
}
