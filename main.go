package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
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
type setlistItem struct {
	EventDate string `json:"eventDate"`
	Venue     struct {
		Name string `json:"name"`
		City struct {
			Name    string `json:"name"`
			Country struct {
				Name string `json:"name"`
			} `json:"country"`
		} `json:"city"`
	} `json:"venue"`
	URL  string `json:"url"`
	Sets struct {
		Set []struct {
			Song []struct {
				Name string `json:"name"`
			} `json:"song"`
		} `json:"set"`
	} `json:"sets"`
}

// Parse une date au format DD-MM-YYYY.
func parserDate(dateStr string) (time.Time, error) {
	return time.Parse("02-01-2006", dateStr)
}

// récupère les dates de concerts depuis l'API relation.
func recupererDatesConcerts(id int) (*Concert, *Concert, error) {
	urlAPI := "https://groupietrackers.herokuapp.com/api/relation/" + strconv.Itoa(id)

	resp, err := http.Get(urlAPI)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, errors.New("appel API relation non OK")
	}

	var relation relationAPI
	if err := json.NewDecoder(resp.Body).Decode(&relation); err != nil {
		return nil, nil, err
	}

	// Collecter toutes les dates avec leurs lieux.
	type dateLieu struct {
		date    time.Time
		lieu    string
		dateStr string
	}

	var toutesDates []dateLieu
	maintenant := time.Now()

	for lieu, dates := range relation.DatesLocations {
		for _, dateStr := range dates {
			date, err := parserDate(dateStr)
			if err != nil {
				log.Printf("erreur parsing date %s : %v", dateStr, err)
				continue
			}
			toutesDates = append(toutesDates, dateLieu{
				date:    date,
				lieu:    lieu,
				dateStr: dateStr,
			})
		}
	}

	if len(toutesDates) == 0 {
		return nil, nil, nil
	}

	// Trier par date.
	sort.Slice(toutesDates, func(i, j int) bool {
		return toutesDates[i].date.Before(toutesDates[j].date)
	})

	// Trouver le prochain concert (première date future).
	var prochain *Concert
	for _, dl := range toutesDates {
		if dl.date.After(maintenant) {
			prochain = &Concert{
				Date:        dl.dateStr,
				Lieu:        dl.lieu,
				EstProchain: true,
			}
			break
		}
	}

	// Trouver le dernier concert (dernière date passée).
	var dernier *Concert
	for i := len(toutesDates) - 1; i >= 0; i-- {
		dl := toutesDates[i]
		if dl.date.Before(maintenant) || dl.date.Equal(maintenant) {
			dernier = &Concert{
				Date:        dl.dateStr,
				Lieu:        dl.lieu,
				EstProchain: false,
			}
			break
		}
	}

	return prochain, dernier, nil
}
