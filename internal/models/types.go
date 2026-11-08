package models

// Tipos de movimiento y de categoría admitidos por la aplicación.
const (
	TypeIncome  = "income"
	TypeExpense = "expense"
)

// IsValidType indica si el valor corresponde a un tipo de movimiento válido.
func IsValidType(value string) bool {
	return value == TypeIncome || value == TypeExpense
}
