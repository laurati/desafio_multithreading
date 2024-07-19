package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

type AddressViaCep struct {
	Cep        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Localidade string `json:"localidade"`
}

type AddressBrasilApi struct {
	CEP          string `json:"cep"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
	Service      string `json:"service"`
}

func GetAddressService(cep string) (interface{}, error) {

	c1 := make(chan AddressViaCep)
	c2 := make(chan AddressBrasilApi)
	errCh := make(chan error, 2)

	urlViaCep := fmt.Sprintf(`http://viacep.com.br/ws/%s/json`, cep)
	urlBrasilApi := fmt.Sprintf(`http://brasilapi.com.br/api/cep/v1/%s`, cep)

	go func() {
		req, err := http.Get(urlViaCep)
		if err != nil {
			errCh <- fmt.Errorf("request err: %v", err)
			return
		}
		defer req.Body.Close()

		res, err := io.ReadAll(req.Body)
		if err != nil {
			errCh <- fmt.Errorf("response err: %v", err)
			return
		}
		var addressViaCep AddressViaCep
		err = json.Unmarshal(res, &addressViaCep)
		if err != nil {
			errCh <- fmt.Errorf("err parsing response: %v", err)
			return
		}
		c1 <- addressViaCep
	}()

	go func() {
		req, err := http.Get(urlBrasilApi)
		if err != nil {
			errCh <- fmt.Errorf("request err: %v", err)
			return
		}
		defer req.Body.Close()

		res, err := io.ReadAll(req.Body)
		if err != nil {
			errCh <- fmt.Errorf("response err: %v", err)
			return
		}
		var addressBrasilApi AddressBrasilApi
		err = json.Unmarshal(res, &addressBrasilApi)
		if err != nil {
			errCh <- fmt.Errorf("err parsing response: %v", err)
			return
		}
		c2 <- addressBrasilApi
	}()

	select {
	case msg := <-c1:
		fmt.Println("addressViaCep ", msg)
		return msg, nil
	case msg := <-c2:
		fmt.Println("addressBrasilApi ", msg)
		return msg, nil
	case err := <-errCh:
		log.Println(err)
		return nil, err
	case <-time.After(time.Second):
		fmt.Println("timeout")
		return nil, fmt.Errorf("timeout")
	}

}

func GetCep(w http.ResponseWriter, r *http.Request) {
	cep := chi.URLParam(r, "cep")
	if cep == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	address, err := GetAddressService(cep)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(address)
}

func main() {
	r := chi.NewRouter()

	r.Get("/{cep}", GetCep)

	http.ListenAndServe(":8084", r)
}
