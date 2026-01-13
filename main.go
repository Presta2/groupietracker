package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
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
	Genre           string       `json:"genre"`
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
			Genre:         "",
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

	// Trouver le prochain concert la première date future.
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

	// Trouver le dernier concert la dernière date passé.
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

// Utilise la variable d'environnement SETLIST_API_KEY pour la clé.
func recupererSetlistPourArtiste(nom string) (*SetlistInfo, error) {
	apiKey := os.Getenv("SETLIST_API_KEY")

	if apiKey == "" {
		apiKey = "kr25FuG2MKExIURHl_oeJRfmdwVkHt5qfKl_"
		log.Printf("⚠️ SETLIST_API_KEY non définie - utilisation clé hardcodée (temporaire)")
	}
	log.Printf("🔍 Recherche setlist pour: %s", nom)

	baseURL := "https://api.setlist.fm/rest/1.0/search/setlists"
	params := url.Values{}
	params.Set("artistName", nom)
	params.Set("p", "1")

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("appel Setlist.fm non OK pour %s : %d", nom, resp.StatusCode)
		return nil, nil
	}

	var resultat setlistSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&resultat); err != nil {
		return nil, err
	}
	if len(resultat.Setlist) == 0 {
		log.Printf(" Aucune setlist trouvée pour %s", nom)
		return nil, nil
	}
	log.Printf(" Setlist trouvée pour %s : %d résultat(s)", nom, len(resultat.Setlist))

	sl := resultat.Setlist[0]

	// Récupérer quelques titres de la setlist.
	chansons := make([]string, 0, 12)
	for _, s := range sl.Sets.Set {
		for _, song := range s.Song {
			if song.Name != "" {
				chansons = append(chansons, song.Name)
				if len(chansons) >= 12 {
					break
				}
			}
		}
		if len(chansons) >= 12 {
			break
		}
	}

	lieu := sl.Venue.Name
	if sl.Venue.City.Name != "" {
		lieu = fmt.Sprintf("%s — %s", sl.Venue.Name, sl.Venue.City.Name)
	}

	info := &SetlistInfo{
		Date:      sl.EventDate,
		Lieu:      lieu,
		Chansons:  chansons,
		SourceURL: sl.URL,
	}
	return info, nil
}

// récupère un artiste par son ID en filtrant l'API.
func recupererArtisteParID(id int) (*Artiste, error) {
	artistes, err := recupererArtistesAPI()
	if err != nil {
		return nil, err
	}
	for _, a := range artistes {
		if a.ID == id {
			// Récupérer les dates de concerts.
			prochain, dernier, err := recupererDatesConcerts(id)
			if err != nil {
				log.Printf("erreur récupération dates concerts pour artiste %d : %v", id, err)
				// On continue même si les dates échouent.
			}
			a.ProchainConcert = prochain
			a.DernierConcert = dernier

			setlist, err := recupererSetlistPourArtiste(a.Nom)
			if err != nil {
				log.Printf("erreur récupération setlist pour artiste %s : %v", a.Nom, err)
			}
			a.Setlist = setlist

			return &a, nil
		}
	}
	return nil, errors.New("artiste introuvable")
}

// Prépare les templates HTML
var (
	modeleAccueil  = template.Must(template.ParseFiles("index.html"))
	modeleArtistes = template.Must(template.ParseFiles("Artists.html"))
	modeleDetail   = template.Must(template.ParseFiles("ArtisteDetail.html"))
)

// Handler page d'accueil.
func pageAccueil(w http.ResponseWriter, r *http.Request) {
	if err := modeleAccueil.Execute(w, nil); err != nil {
		log.Printf("erreur rendu accueil : %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
}

// page artistes (rendu HTML avec la liste).
func pageArtistes(w http.ResponseWriter, r *http.Request) {
	artistes, err := recupererArtistesAPI()
	if err != nil {
		log.Printf("erreur récupération API artistes : %v", err)
		http.Error(w, "Impossible de charger les artistes", http.StatusInternalServerError)
		return
	}

	donnees := struct {
		Artistes []Artiste
	}{
		Artistes: artistes,
	}

	if err := modeleArtistes.Execute(w, donnees); err != nil {
		log.Printf("erreur rendu artistes : %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
}

// page détail d'un artiste.
func pageArtisteDetail(w http.ResponseWriter, r *http.Request) {

	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) != 2 {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.Atoi(segments[1])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	artiste, err := recupererArtisteParID(id)
	if err != nil {
		log.Printf("erreur récupération artiste %d : %v", id, err)
		http.Error(w, "Artiste introuvable", http.StatusNotFound)
		return
	}

	donnees := struct {
		Artiste *Artiste
	}{
		Artiste: artiste,
	}

	if err := modeleDetail.Execute(w, donnees); err != nil {
		log.Printf("erreur rendu détail artiste : %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
}

// Endpoint JSON simple pour tester rapidement depuis le navigateur.
func apiArtistes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	artistes, err := recupererArtistesAPI()
	if err != nil {
		log.Printf("erreur récupération API artistes : %v", err)
		http.Error(w, "Impossible de charger les artistes", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(artistes); err != nil {
		log.Printf("erreur réponse JSON : %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
}

func main() {
	mux := http.NewServeMux()

	// Fichiers statiques (CSS, images...).
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Pages principales.
	mux.HandleFunc("/", pageAccueil)
	mux.HandleFunc("/artistes", pageArtistes)
	mux.HandleFunc("/artistes/", pageArtisteDetail)

	// Petite API JSON.
	mux.HandleFunc("/api/artistes", apiArtistes)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Serveur démarré sur http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("serveur stoppé : %v", err)
	}
}
