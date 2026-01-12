package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Artiste représente les infos qu'on affichera côté front.
type Artiste struct {
	ID              int          `json:"id"`
	Nom             string       `json:"nom"`
	Image           string       `json:"image"`
	Membres         []string     `json:"membres"`
	AnneeCreation   int          `json:"annee_creation"`
	PremierAlbum    string       `json:"premier_album"`
	Genre           string       `json:"genre"` // champ non fourni par l'API, restera vide
	ProchainConcert *Concert     `json:"prochain_concert"`
	DernierConcert  *Concert     `json:"dernier_concert"`
	Setlist         *SetlistInfo `json:"setlist"`
}

// Concert représente une date de concert avec lieu.
type Concert struct {
	Date        string `json:"date"`
	Lieu        string `json:"lieu"`
	EstProchain bool   `json:"est_prochain"`
}

// SetlistInfo représente une setlist simplifiée provenant de l'API Setlist.fm.
type SetlistInfo struct {
	Date      string   `json:"date"`
	Lieu      string   `json:"lieu"`
	Chansons  []string `json:"chansons"`
	SourceURL string   `json:"source_url"`
}

// Réponse brute de l'API (champs en anglais).
type artisteAPI struct {
	ID           int      `json:"id"`
	Nom          string   `json:"name"`
	Image        string   `json:"image"`
	Membres      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	PremierAlbum string   `json:"firstAlbum"`
}

// Réponse brute de l'API (champs en anglais).
type artisteAPI struct {
	ID           int      `json:"id"`
	Nom          string   `json:"name"`
	Image        string   `json:"image"`
	Membres      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	PremierAlbum string   `json:"firstAlbum"`
}

// récupère les artistes depuis l'API publique.
func recupererArtistesAPI() ([]Artiste, error) {
	const urlAPI = "https://groupietrackers.herokuapp.com/api/artists"

	resp, err := http.Get(urlAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("appel API artistes non OK")
	}

	var bruts []artisteAPI
	if err := json.NewDecoder(resp.Body).Decode(&bruts); err != nil {
		return nil, err
	}

	artistes := make([]Artiste, 0, len(bruts))
	for _, a := range bruts {
		artistes = append(artistes, Artiste{
			ID:            a.ID,
			Nom:           a.Nom,
			Image:         a.Image,
			Membres:       a.Membres,
			AnneeCreation: a.CreationDate,
			PremierAlbum:  a.PremierAlbum,
			Genre:         "", // l'API ne fournit pas le genre
		})
	}
	return artistes, nil
}

// Structure pour l'API relation (dates et lieux).
type relationAPI struct {
	DatesLocations map[string][]string `json:"datesLocations"`
}

// Structures pour l'API Setlist.fm (recherche de setlists par nom d'artiste).
type setlistSearchResponse struct {
	Setlist []setlistItem `json:"setlist"`
}
