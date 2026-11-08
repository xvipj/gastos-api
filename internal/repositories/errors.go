// Package repositories contiene el acceso a datos con GORM. Ninguna consulta
// de este paquete conoce detalles de HTTP ni de la lógica de negocio.
package repositories

import "errors"

// ErrTokenAlreadyRevoked indica que una rotación perdió una carrera contra
// otra petición que ya había revocado el mismo refresh token.
var ErrTokenAlreadyRevoked = errors.New("refresh token ya revocado")
