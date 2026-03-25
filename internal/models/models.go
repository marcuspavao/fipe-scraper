package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// TabelaReferencia — collection "TabelaReferencia"
// Um documento por mês de referência da FIPE
type TabelaReferencia struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Codigo int                `bson:"codigo"` // ex: 319
	Mes    string             `bson:"mes"`    // ex: "março/2026"
}

// YearData — preço de um veículo em um ano específico
// Ano 32000 = 0km (zero quilômetro)
type YearData struct {
	YearCode       string `bson:"yearCode"`       // ex: "2022-1"
	Year           int    `bson:"year"`           // ex: 2022 ou 32000 (0km)
	Price          string `bson:"price"`          // ex: "R$ 52.100,00"
	MonthReference string `bson:"monthReference"` // ex: "janeiro de 2026 "
}

// ModelData — um modelo com seus anos/preços
type ModelData struct {
	ModelCode int        `bson:"modelCode"` // ex: 4921
	ModelName string     `bson:"modelName"` // ex: "Argo Drive 1.0"
	Years     []YearData `bson:"years"`
}

// VeiculoDoc — collection "Veiculos"
// UM documento por marca por tabela de referência
// Compatível com fipe_project
type VeiculoDoc struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	MonthYearID int                `bson:"monthYearId"` // código da tabela (ex: 319)
	BrandCode   int                `bson:"brandCode"`   // código da marca
	BrandName   string             `bson:"brandName"`   // nome da marca (ex: "Fiat")
	Models      []ModelData        `bson:"models"`
}

// — structs temporárias para deserializar a API da FIPE —

type FIPETabela struct {
	Codigo int    `json:"Codigo"`
	Mes    string `json:"Mes"`
}

type FIPEMarca struct {
	Value string `json:"Value"`
	Label string `json:"Label"`
}

type FIPEModelo struct {
	Value int    `json:"Value"`
	Label string `json:"Label"`
}

type FIPEAnoModelo struct {
	Value string `json:"Value"` // ex: "2022-1"
	Label string `json:"Label"` // ex: "2022 Gasolina"
}

type FIPEModelosResp struct {
	Modelos []FIPEModelo    `json:"Modelos"`
	Anos    []FIPEAnoModelo `json:"Anos"`
}

type FIPEValorResp struct {
	Marca         string `json:"Marca"`
	Modelo        string `json:"Modelo"`
	AnoModelo     int    `json:"AnoModelo"`
	CodigoFipe    string `json:"CodigoFipe"`
	Valor         string `json:"Valor"`         // FIPE API retorna "Valor" em vez de "Preco"
	MesReferencia string `json:"MesReferencia"` // Referência do mês
}
