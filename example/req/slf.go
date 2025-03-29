package req

import "github.com/viant/mly/shared/client"

type SLF struct {
	SA  string
	SL  string
	Aux string
}

func (s *SLF) ToMessage(msg *client.Message) error {
	msg.StringKey("sa", s.SA)
	msg.StringKey("sl", s.SL)
	msg.StringKey("aux", s.Aux)

	return nil
}
