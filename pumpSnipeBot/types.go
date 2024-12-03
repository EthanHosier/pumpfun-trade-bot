package pumpSnipeBot

type BotError struct {
	error     error
	forceQuit bool // if not force quit, will just log and continue
}
