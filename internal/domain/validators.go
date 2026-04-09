package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// CommonPasswords — Lista mínima de contraseñas inseguras que deben rechazarse.
var CommonPasswords = map[string]bool{
	"password":     true,
	"12345678":     true,
	"123456789":    true,
	"password123":  true,
	"admin":        true,
	"admin123":     true,
	"letmein":      true,
	"welcome":      true,
	"qwerty123":    true,
	"abc123456":    true,
	"integritypos": true,
	"pos123456":    true,
	"prueba12345":  true,
	"test123456":   true,
	"integrity123": true,
}

// ValidatePassword verifica que una contraseña cumpla con los requisitos mínimos de seguridad:
// - Al menos 8 caracteres
// - Al menos 1 letra mayúscula
// - Al menos 1 letra minúscula
// - Al menos 1 número
// - Al menos 1 carácter especial
// - No puede ser una contraseña común
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	var missing []string
	if !hasUpper {
		missing = append(missing, "one uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "one lowercase letter")
	}
	if !hasNumber {
		missing = append(missing, "one number")
	}
	if !hasSpecial {
		missing = append(missing, "one special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("password must contain at least %s", strings.Join(missing, ", "))
	}

	// Check against common passwords
	if CommonPasswords[strings.ToLower(password)] {
		return fmt.Errorf("password is too common, please choose a more secure password")
	}

	return nil
}

// ValidateUsername verifica que un nombre de usuario sea válido:
// - Entre 3 y 50 caracteres
// - Solo letras, números, guiones y guiones bajos
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	if len(username) > 50 {
		return fmt.Errorf("username must be at most 50 characters long")
	}

	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && char != '-' && char != '_' {
			return fmt.Errorf("username can only contain letters, numbers, hyphens and underscores")
		}
	}

	return nil
}
