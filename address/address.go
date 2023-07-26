package address

import (
	"encoding/json"

	"github.com/xssnick/tonutils-go/address"
)

type Address struct {
	*address.Address
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	addr, err := address.ParseAddr(s)
	if err != nil {
		return err
	}

	a.Address = addr

	return nil
}
