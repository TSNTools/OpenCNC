package stream

import "fmt"

func (x *StreamId) AsKey() string {
	return fmt.Sprintf("%s-%d", x.GetMacAddress(), x.GetUniqueId())
}
