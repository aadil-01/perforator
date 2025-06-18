package querylang

import (
	"fmt"
)

func (f *Selector) IsEmpty() bool {
	return len(f.Matchers) == 0
}

func (f *Selector) AllMentionedFields() []string {
	fields := make(map[string]struct{})
	for _, m := range f.Matchers {
		fields[m.Field] = struct{}{}
	}
	return mapKeys(fields)
}

func (f *Selector) UniqueFieldValues(field string) []Value {
	values := make(map[Value]struct{})
	for _, m := range f.Matchers {
		if m.Field == field {
			for _, cond := range m.Conditions {
				values[cond.Value] = struct{}{}
			}
		}
	}
	return mapKeys(values)
}

// StrictMap returns a field-value mapping of Selector
// if there are only strict equality operators are used
// and any field corresponds to exactly 1 string value.
func (f *Selector) StrictMap() (map[string]string, error) {
	result := make(map[string]string, len(f.Matchers))
	for _, m := range f.Matchers {
		for _, c := range m.Conditions {
			if !c.IsStrictEq() {
				return nil, fmt.Errorf("field `%s` is involved in non-strict-equality comparison", m.Field)
			}
			if _, present := result[m.Field]; present {
				return nil, fmt.Errorf("found multiple values correspond to field `%s`", m.Field)
			}
			if sv, ok := c.Value.(String); !ok {
				return nil, fmt.Errorf("found non-string literal comparison with field `%s`", m.Field)
			} else {
				result[m.Field] = sv.Value
			}
		}
	}
	return result, nil
}

func (f *Selector) ReplaceConditionValue(field string, oldValue Value, newValues []Value) {
	for _, matcher := range f.Matchers {
		if matcher.Field == field {
			exists := hasCondition(matcher.Conditions, func(item *Condition) bool {
				return item.Value == oldValue
			})

			if !exists {
				continue
			}

			var newConditions []*Condition
			for _, condition := range matcher.Conditions {
				if condition.Value == oldValue {
					for _, newValue := range newValues {
						newConditions = append(newConditions, &Condition{
							Operator: condition.Operator,
							Inverse:  condition.Inverse,
							Value:    newValue,
						})
					}
				}
			}

			matcher.Conditions = filterConditions(matcher.Conditions, func(item *Condition, _ int) bool {
				return item.Value != oldValue
			})

			matcher.Conditions = append(matcher.Conditions, newConditions...)
		}
	}
}
