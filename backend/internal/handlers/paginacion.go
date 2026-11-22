package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Los listados de reportes son paginados: el cliente pide `pagina` y
// `por_pagina` y recibe el sobre comun (items/total/pagina/por_pagina/paginas).
const (
	PorPaginaPorDefecto = 20
	PorPaginaMaximo     = 100
)

// leerPaginacion toma `pagina` y `por_pagina` del query y los acota.
func leerPaginacion(c *gin.Context) (pagina, porPagina int) {
	pagina, _ = strconv.Atoi(c.DefaultQuery("pagina", "1"))
	porPagina, _ = strconv.Atoi(c.DefaultQuery("por_pagina", strconv.Itoa(PorPaginaPorDefecto)))
	if pagina < 1 {
		pagina = 1
	}
	if porPagina < 1 {
		porPagina = PorPaginaPorDefecto
	}
	if porPagina > PorPaginaMaximo {
		porPagina = PorPaginaMaximo
	}
	return pagina, porPagina
}

// respuestaPagina arma el sobre comun de los listados paginados.
func respuestaPagina(c *gin.Context, items any, total int64, pagina, porPagina int) {
	paginas := int((total + int64(porPagina) - 1) / int64(porPagina))
	if paginas < 1 {
		paginas = 1
	}
	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"total":      total,
		"pagina":     pagina,
		"por_pagina": porPagina,
		"paginas":    paginas,
	})
}

// contarPorUsuario suma el total de filas de un modelo para un usuario.
func contarPorUsuario(db *gorm.DB, modelo any, userID uint) int64 {
	var total int64
	_ = db.Model(modelo).Where("user_id = ?", userID).Count(&total).Error
	return total
}
