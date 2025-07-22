package querylang

import (
	"errors"
	"strings"

	"github.com/yandex/perforator/observability/lib/querylang/operator"
)

func (c *Condition) IsStrictEq() bool {
	return c.Operator == operator.Eq && !c.Inverse
}

func (c *Condition) IsEqOrNotEqOrExists() bool {
	return c.Operator == operator.Eq || c.Operator == operator.Exists
}

var errSmartquoteNotApplicable = errors.New("smartquote not applicable")

func smartquote(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s) + 2)
	if !strings.ContainsRune(s, '"') {
		b.WriteRune('"')
		b.WriteString(s)
		b.WriteRune('"')
		return b.String(), nil
	}
	if !strings.ContainsRune(s, '\'') {
		b.WriteRune('\'')
		b.WriteString(s)
		b.WriteRune('\'')
		return b.String(), nil
	}
	return "", errSmartquoteNotApplicable
}
