package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/marcuspavao/fipe-scraper/internal/models"
)

const (
	baseURL          = "https://veiculos.fipe.org.br/api/veiculos"
	tipoVeiculoCarro = 1
)

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

func post(endpoint string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("serializar payload: %w", err)
	}

	maxRetries := 3
	backoff := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("POST", baseURL+"/"+endpoint, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("criar requisição: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Referer", "https://veiculos.fipe.org.br/")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := httpClient.Do(req)
		if err != nil {
			if i < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, fmt.Errorf("executar requisição: %w", err)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if i < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return bodyBytes, nil
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			fmt.Printf("⚠️ %d Forbidden/TooMany na FIPE (%s), aguardando %v...\n", resp.StatusCode, endpoint, backoff)
			if i < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
		}

		return nil, fmt.Errorf("status inesperado da FIPE: %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("máximo de tentativas atingido para %s", endpoint)
}

// BuscarTabelas retorna os meses de referência disponíveis na FIPE
func BuscarTabelas() ([]models.TabelaReferencia, error) {
	data, err := post("ConsultarTabelaDeReferencia", map[string]any{})
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Codigo int    `json:"Codigo"`
		Mes    string `json:"Mes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("deserializar tabelas: %w", err)
	}

	tabelas := make([]models.TabelaReferencia, 0, len(raw))
	for _, r := range raw {
		tabelas = append(tabelas, models.TabelaReferencia{
			Codigo: r.Codigo,
			Mes:    r.Mes,
		})
	}
	return tabelas, nil
}

// BuscarMarcas retorna as marcas disponíveis para uma tabela de referência
func BuscarMarcas(codigoTabela int) ([]models.FIPEMarca, error) {
	data, err := post("ConsultarMarcas", map[string]any{
		"codigoTabelaReferencia": codigoTabela,
		"codigoTipoVeiculo":      tipoVeiculoCarro,
	})
	if err != nil {
		return nil, err
	}

	var marcas []models.FIPEMarca
	if err := json.Unmarshal(data, &marcas); err != nil {
		return nil, fmt.Errorf("deserializar marcas: %w", err)
	}
	return marcas, nil
}

// BuscarModelos retorna os modelos e anos disponíveis para uma marca
func BuscarModelos(codigoTabela, codigoMarca int) (*models.FIPEModelosResp, error) {
	data, err := post("ConsultarModelos", map[string]any{
		"codigoTabelaReferencia": codigoTabela,
		"codigoTipoVeiculo":      tipoVeiculoCarro,
		"codigoMarca":            codigoMarca,
	})
	if err != nil {
		return nil, err
	}

	var resp models.FIPEModelosResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("deserializar modelos: %w", err)
	}
	return &resp, nil
}

// BuscarAnos retorna os anos disponíveis para um modelo específico
func BuscarAnos(codigoTabela, codigoMarca, codigoModelo int) ([]models.FIPEAnoModelo, error) {
	data, err := post("ConsultarAnoModelo", map[string]any{
		"codigoTabelaReferencia": codigoTabela,
		"codigoTipoVeiculo":      tipoVeiculoCarro,
		"codigoMarca":            codigoMarca,
		"codigoModelo":           codigoModelo,
	})
	if err != nil {
		return nil, err
	}

	var anos []models.FIPEAnoModelo
	if err := json.Unmarshal(data, &anos); err != nil {
		return nil, fmt.Errorf("deserializar anos: %w", err)
	}
	return anos, nil
}

// BuscarValor consulta o preço de um veículo específico na FIPE
func BuscarValor(codigoTabela, codigoMarca, codigoModelo, anoModelo, codigoTipoCombustivel int) (*models.FIPEValorResp, error) {
	data, err := post("ConsultarValorComTodosParametros", map[string]any{
		"codigoTabelaReferencia": codigoTabela,
		"codigoTipoVeiculo":      tipoVeiculoCarro,
		"codigoMarca":            codigoMarca,
		"codigoModelo":           codigoModelo,
		"anoModelo":              anoModelo,
		"codigoTipoCombustivel":  codigoTipoCombustivel,
		"tipoConsulta":           "tradicional",
	})
	if err != nil {
		return nil, err
	}

	var resp models.FIPEValorResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("deserializar valor: %w", err)
	}
	return &resp, nil
}

// ParseAnoModelo converte "2022-1" → ano=2022, combustivel=1
func ParseAnoModelo(value string) (ano, combustivel int, err error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("formato inválido: %s", value)
	}
	ano, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("ano inválido: %s", parts[0])
	}
	combustivel, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("combustível inválido: %s", parts[1])
	}
	return
}

// ParsePreco converte "R$ 45.870,00" → 45870.00
func ParsePreco(precoStr string) float64 {
	s := strings.TrimSpace(precoStr)
	s = strings.ReplaceAll(s, "R$", "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}
