package domain

import "fmt"

type Money int64

type Quantity int64

func (m Money) String() string {
	return fmt.Sprintf("$%.2f", float64(m)/100.0)
}

func (m Money) Add(other Money) Money {
	return m + other
}

func (m Money) Sub(other Money) Money {
	return m - other
}

func (m Money) IsZero() bool {
	return m == 0
}

func (m Money) IsNegative() bool {
	return m < 0
}

func (q Quantity) Int64() int64 {
	return int64(q)
}
