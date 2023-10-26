package address

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/xssnick/tonutils-go/address"
)

type Address struct {
	*address.Address
}

func (a *Address) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("%v:%v", a.Workchain(), hex.EncodeToString(a.Data()))
	return json.Marshal(s)
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var wc int
	var hash string
	_, err = fmt.Sscanf(s, "%v:%v", &wc, &hash)
	if err != nil {
		return err
	}

	var wcByte byte
	if wc == 0 {
		wcByte = 0
	} else if wc == -1 {
		wcByte = 0xff
	} else {
		return fmt.Errorf("unknown workchain value: %v", wc)
	}

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return err
	}
	if len(hashBytes) != 32 {
		return fmt.Errorf("incorrect address hash part length: %v", len(hashBytes))
	}

	a.Address = address.NewAddress(0, wcByte, hashBytes)

	return nil
}
