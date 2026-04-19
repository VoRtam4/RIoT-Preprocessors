/**
 * @File: loader.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: Načítání jednotlivých částí TMC tabulky.
 */

package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"golang.org/x/text/encoding/charmap"
)

// --------------------
// ROADS
// --------------------

// RoadSegmentNode představuje silnici nebo její segment, včetně podsegmentů, kdy pole Segments může mít další RoadSegmentNode.
type RoadSegmentNode struct {
	LCD        string              `json:"lcd"`
	RoadNumber string              `json:"road_number"`
	RoadName   string              `json:"road_name"`
	FirstName  string              `json:"firstname"`
	SecondName string              `json:"secondname"`
	AreaRef    string              `json:"area_ref"`
	Segments   []*RoadSegmentNode  `json:"segments,omitempty"`
}

// Funkce BuildRoadMap načte silnice i segmenty a vrátí mapu LCD → RoadSegmentNode.
func BuildRoadMap(baseDir string) (map[string]*RoadSegmentNode, error) {
	roadMap := make(map[string]*RoadSegmentNode)

	// Načti silnice
	rf, err := os.Open(filepath.Join(baseDir, "ltcze10_1_roads.txt"))
	if err != nil {
		return nil, fmt.Errorf("chyba při načítání silnic: %v", err)
	}
	defer rf.Close()

	// Dekodér CP1250→UTF8.
	decoder := charmap.Windows1250.NewDecoder()
	rReader := decoder.Reader(rf)

	r := csv.NewReader(rReader)
	r.Comma = ';'
	r.FieldsPerRecord = -1

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 11 {
			continue
		}
		lcd := strings.TrimSpace(rec[2])
		node := &RoadSegmentNode{
			LCD:        lcd,
			RoadNumber: strings.TrimSpace(rec[6]),
			RoadName:   strings.TrimSpace(rec[7]),
			FirstName:  strings.TrimSpace(rec[8]),
			SecondName: strings.TrimSpace(rec[9]),
			AreaRef:    strings.TrimSpace(rec[10]),
		}
		roadMap[lcd] = node
	}

	// Načti segmenty.
	sf, err := os.Open(filepath.Join(baseDir, "ltcze10_1_segments.txt"))
	if err != nil {
		return nil, fmt.Errorf("chyba při načítání segmentů: %v", err)
	}
	defer sf.Close()

	segDecoder := charmap.Windows1250.NewDecoder()
	segReader := segDecoder.Reader(sf)

	sr := csv.NewReader(segReader)
	sr.Comma = ';'
	sr.FieldsPerRecord = -1

	for {
		rec, err := sr.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 13 {
			continue
		}
		segLCD := strings.TrimSpace(rec[2])
		// ROA_LCD
		parentID := strings.TrimSpace(rec[12])
		segNode := &RoadSegmentNode{
			LCD:        segLCD,
			RoadNumber: strings.TrimSpace(rec[6]),
			RoadName:   strings.TrimSpace(rec[7]),
			FirstName:  strings.TrimSpace(rec[8]),
			SecondName: strings.TrimSpace(rec[9]),
			AreaRef:    strings.TrimSpace(rec[10]),
		}
		// Připoj segment k nadřazené silnici.
		if parent, ok := roadMap[parentID]; ok {
			parent.Segments = append(parent.Segments, segNode)
		}
		roadMap[segLCD] = segNode
	}

	return roadMap, nil
}

// BuildRoadList vrátí plochý seznam silnic (včetně podsegmentů) pro danou oblast a její podlokality.
func BuildRoadList(baseDir, areaRef string) ([]*RoadSegmentNode, error) {
	roadMap, err := BuildRoadMap(baseDir)
	if err != nil {
		return nil, err
	}

	// Získej všechny podlokality.
	if areaRef == "" {
		areaRef = "1"
	}
	subs := GetAllSubLocations(areaRef)
	allowed := make(map[string]bool)
	for _, a := range subs {
		allowed[a] = true
	}

	// Urči, které LCD jsou segmenty.
	segmentIDs := make(map[string]bool)
	for _, node := range roadMap {
		for _, seg := range node.Segments {
			segmentIDs[seg.LCD] = true
		}
	}

	// Vytvoř výsledný seznam pouze top-level silnic (bez samostatných segmentů).
	var list []*RoadSegmentNode
	for _, node := range roadMap {
		if !segmentIDs[node.LCD] && allowed[node.AreaRef] {
			list = append(list, node)
		}
	}
	return list, nil
}


// --------------------
// POINTS
// --------------------

// Point je mapování názvu sloupce → hodnota pro jeden řádek.
type Point map[string]string

// Loader drží načtené řádky.
type Loader struct {
	baseDir string
	Points  []Point
}

// Funkce NewLoader inicializuje loader s cestou k datům.
func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

// Uchovává strukturu admins.txt.
type AdminNode struct {
	LCD      string      `json:"lcd"`
	Name     string      `json:"firstname"`
	Children []AdminNode `json:"children,omitempty"`
}

// Load načte celý soubor points.txt do Loader.Points.
func (l *Loader) Load() error {
	fpath := filepath.Join(l.baseDir, "ltcze10_1_points.txt")
	f, err := os.Open(fpath)
	if err != nil {
		return fmt.Errorf("open points file: %w", err)
	}
	defer f.Close()

	// Zabalíme reader do CP1250→UTF8 dekodéru.
	decoder := charmap.Windows1250.NewDecoder()
	reader := decoder.Reader(f)

	r := csv.NewReader(reader)
	r.Comma = ';'
	r.LazyQuotes = true

	headers, err := r.Read()
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		pt := make(Point)
		for i, h := range headers {
			if i < len(record) {
				pt[h] = record[i]
			}
		}
		l.Points = append(l.Points, pt)
	}
	return nil
}

// --------------------
// ADMINS
// --------------------

// AdminLocation reprezentuje jednu lokalitu z admins.txt.
type AdminLocation struct {
	CID       string `json:"CID"`
	TABCD     string `json:"TABCD"`
	LCD       string `json:"LCD"`
	CLASS     string `json:"CLASS"`
	TCD       string `json:"TCD"`
	STCD      string `json:"STCD"`
	FIRSTNAME string `json:"FIRSTNAME"`
	AREA_REF  string `json:"AREA_REF"`
	AREA_NAME string `json:"AREA_NAME"`
}

// AdminsTree mapuje LCD → seznam podřízených LCD.
var AdminsTree = make(map[string][]string)

// AdminsMap obsahuje LCD → detailní záznam.
var AdminsMap = make(map[string]AdminLocation)

// BuildAdminTree vrátí celý strom lokalit (rekurzivně).
func BuildAdminTree() []AdminNode {
	var build func(parent string) []AdminNode
	build = func(parent string) []AdminNode {
		childrenLCDs := AdminsTree[parent]
		var nodes []AdminNode
		for _, lcd := range childrenLCDs {
			loc := AdminsMap[lcd]
			nodes = append(nodes, AdminNode{
				LCD:      lcd,
				Name:     loc.FIRSTNAME,
				Children: build(lcd),
			})
		}
		return nodes
	}

	// Strom začíná od LCD="1" (Evropa).
	if _, exists := AdminsMap["1"]; !exists {
		return []AdminNode{}
	}
	root := AdminsMap["1"]
	return []AdminNode{{
		LCD:      root.LCD,
		Name:     root.FIRSTNAME,
		Children: build("1"),
	}}
}

// Funkce LoadAdminLocations načte ltcze10_1_admins.txt a vytvoří strom.
func LoadAdminLocations(path string) error {
	fpath := filepath.Join(path, "ltcze10_1_admins.txt")
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := charmap.Windows1250.NewDecoder()
	reader := decoder.Reader(f)

	r := csv.NewReader(reader)
	r.Comma = ';'
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil {
		return err
	}

	for i, row := range records {
		if i == 0 || len(row) < 9 {
			continue
		}
		loc := AdminLocation{
			CID:       row[0],
			TABCD:     row[1],
			LCD:       row[2],
			CLASS:     row[3],
			TCD:       row[4],
			STCD:      row[5],
			FIRSTNAME: row[6],
			AREA_REF:  row[7],
			AREA_NAME: row[8],
		}
		AdminsMap[loc.LCD] = loc
		if loc.AREA_REF != "" {
			AdminsTree[loc.AREA_REF] = append(AdminsTree[loc.AREA_REF], loc.LCD)
		}
	}
	return nil
}

// GetAllSubLocations vrátí LCD včetně všech zanořených podlokalit.
func GetAllSubLocations(lcd string) []string {
	result := []string{}
	queue := []string{lcd}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children := AdminsTree[current]
		result = append(result, children...)
		queue = append(queue, children...)
	}

	return append([]string{lcd}, result...)
}


// BuildAdminTreeFilteredFromPoints vrátí strom oblastí, ale omezí jej
// jen na větve, kde byly nalezeny zadané silnice/segmenty ve loader.Points.
func BuildAdminTreeFilteredFromPoints(loader *Loader, roadCodes []string) ([]AdminNode, error) {
	// 1) sestav množinu kódů
	codeSet := make(map[string]bool)
	for _, param := range roadCodes {
		for _, code := range strings.Split(param, ",") {
			codeSet[strings.TrimSpace(code)] = true
		}
	}

	// 2) najdi všechny oblasti (AREA_REF) pro body s těmito kódy.
	areaSet := make(map[string]bool)
	for _, p := range loader.Points {
		if codeSet[p["ROA_LCD"]] || codeSet[p["SEG_LCD"]] {
			areaSet[p["AREA_REF"]] = true
		}
	}

	// 3) žádné kódy nebo žádné shody → celý strom.
	if len(codeSet) == 0 || len(areaSet) == 0 {
		return BuildAdminTree(), nil
	}

	// 4) sestav parentMap.
	parentMap := make(map[string]string)
	for parent, kids := range AdminsTree {
		for _, child := range kids {
			parentMap[child] = parent
		}
	}
	// pomocná pro cestu k root.
	getAncestors := func(lcd string) []string {
		path := []string{}
		for cur := lcd; cur != ""; cur = parentMap[cur] {
			path = append([]string{cur}, path...)
			if cur == "1" {
				break
			}
		}
		return path
	}

	// 5) najdi cesty pro každou oblast.
	var paths [][]string
	for area := range areaSet {
		paths = append(paths, getAncestors(area))
	}

	// 6) rekurzivní stavba stromu jen po cestách.
	var build func(string) []AdminNode
	build = func(parent string) []AdminNode {
		var out []AdminNode
		for _, kid := range AdminsTree[parent] {
			// je kid na některé cestě k areaSet?
			onPath := false
			for _, path := range paths {
				for _, step := range path {
					if step == kid {
						onPath = true
						break
					}
				}
				if onPath {
					break
				}
			}
			if !onPath {
				continue
			}
			loc := AdminsMap[kid]
			n := AdminNode{LCD: kid, Name: loc.FIRSTNAME}
			n.Children = build(kid)
			out = append(out, n)
		}
		return out
	}

	// 7) výsledný strom od rootu.
	rootLoc := AdminsMap["1"]
	root := AdminNode{LCD: "1", Name: rootLoc.FIRSTNAME, Children: build("1")}
	return []AdminNode{root}, nil
}
