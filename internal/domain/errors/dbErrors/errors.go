package dbErrors

import "fmt"

type CottageNameDoesNotExist struct {
	CottageName string
}

func (e *CottageNameDoesNotExist) Error() string {
	return fmt.Sprintf("Cottage with name %s does not exist", e.CottageName)
}
