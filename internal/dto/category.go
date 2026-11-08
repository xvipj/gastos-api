package dto

import "time"

// CategoryRequest es el cuerpo para crear o actualizar una categoría.
type CategoryRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100" example:"Alimentación"`
	Type string `json:"type" binding:"required,oneof=income expense" example:"expense"`
}

// CategoryResponse expone una categoría del usuario.
type CategoryResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"Alimentación"`
	Type      string    `json:"type" example:"expense"`
	CreatedAt time.Time `json:"created_at"`
}

// CategoryResponseDoc documenta una categoría en Swagger.
type CategoryResponseDoc struct {
	Data CategoryResponse `json:"data"`
}

// CategoriesResponseDoc documenta un listado de categorías en Swagger.
type CategoriesResponseDoc struct {
	Data []CategoryResponse `json:"data"`
}
