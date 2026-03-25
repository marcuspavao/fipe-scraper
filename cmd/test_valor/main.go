package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{Timeout: 10 * time.Second}
	payload := map[string]interface{}{
		"codigoTipoVeiculo":      1, // Carro
		"codigoTabelaReferencia": 331, // março/2026
		"codigoMarca":            21, // Fiat
		"codigoModelo":           437, // 147 C/ CL
		"anoModelo":              1987,
		"codigoTipoCombustivel":  1,
		"tipoConsulta":           "tradicional",
	}

	valorData, err := post(client, "ConsultarValorComTodosParametros", payload)
	if err != nil {
		fmt.Printf("Error valor: %v\n", err)
		return
	}
	fmt.Printf("Valor Response: %s\n", string(valorData))
}

func post(client *http.Client, endpoint string, payload interface{}) ([]byte, error) {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://veiculos.fipe.org.br/api/veiculos/"+endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://veiculos.fipe.org.br/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
