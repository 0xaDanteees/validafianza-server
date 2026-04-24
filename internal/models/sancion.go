package models

import (
	"time"

	"github.com/google/uuid"
)

// SancionSCG represents a sanction from SCG (CDMX Comptroller)
type SancionSCG struct {
	No              string `json:"no" example:"1"`
	Expediente      string `json:"expediente" example:"SC/DGRALSG/PA/123/2024"`
	Persona         string `json:"persona" example:"EMPRESA EJEMPLO SA DE CV"`
	RFC             string `json:"rfc" example:"EEJ850101XXX"`
	FechaResolucion string `json:"fecha_resolucion" example:"2024-03-15"`
	Infraccion      string `json:"infraccion" example:"Artículo 59 fracción I"`
	Sancion         string `json:"sancion" example:"Inhabilitación temporal"`
	Circular        string `json:"circular" example:"CGCDMX/001/2024"`
	Publicacion     string `json:"publicacion" example:"Gaceta Oficial CDMX 15-03-2024"`
	Vigencia        string `json:"vigencia" example:"2025-12-31"`
}

// SancionSFP represents a sanction from SFP (Public Function)
type SancionSFP struct {
	NumeroExpediente  string `json:"numero_expediente" example:"EXP-SFP-2024-001"`
	NombreRazonSocial string `json:"nombre_razon_social" example:"PROVEEDOR EJEMPLO SA DE CV"`
	RFC               string `json:"rfc" example:"PEJ850101XXX"`
	LeyesInfringidas  string `json:"leyes_infringidas" example:"Ley Federal de Responsabilidades Administrativas"`
	MontoMulta        string `json:"monto_multa" example:"$500,000.00 MXN"`
	PlazoInha         string `json:"plazo_inha" example:"2 años"`
}

// SancionResultSCG is the response for SCG sanction validation
type SancionResultSCG struct {
	RFC        string       `json:"rfc" example:"EEJ850101XXX"`
	Sancionado bool         `json:"sancionado" example:"false"`
	Sanciones  []SancionSCG `json:"sanciones,omitempty"`
	Message    string       `json:"message" example:"Proveedor NO sancionado ante SCG"`
	ValidadoEn time.Time    `json:"validado_en" example:"2024-04-23T20:00:00Z"`
}

// SancionResultSFP is the response for SFP sanction validation
type SancionResultSFP struct {
	RFC        string       `json:"rfc" example:"PEJ850101XXX"`
	Sancionado bool         `json:"sancionado" example:"true"`
	Sanciones  []SancionSFP `json:"sanciones,omitempty"`
	Message    string       `json:"message" example:"Proveedor SANCIONADO ante SFP (Función Pública)"`
	ValidadoEn time.Time    `json:"validado_en" example:"2024-04-23T20:00:00Z"`
}

// SancionResult is the response for RFC sanction validation
type SancionResult struct {
	RFC        string       `json:"rfc" example:"EEJ850101XXX"`
	Sancionado bool         `json:"sancionado" example:"true"`
	SCG        []SancionSCG `json:"scg,omitempty"`
	SFP        []SancionSFP `json:"sfp,omitempty"`
	Message    string       `json:"message,omitempty" example:"Proveedor NO sancionado"`
	ValidadoEn time.Time    `json:"validado_en" example:"2024-04-23T20:00:00Z"`
}

type SancionCache struct {
	ID         uuid.UUID `db:"id"`
	RFC        string    `db:"rfc"`
	Sancionado bool      `db:"sancionado"`
	Detalles   string    `db:"detalles"`
	CreatedAt  time.Time `db:"created_at"`
	ExpiresAt  time.Time `db:"expires_at"`
}
