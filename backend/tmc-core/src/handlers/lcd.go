/**
 * @File: lcd.go
 * @Author: Dominik Vondruška
 * @Project: Bakalářská práce — Systém pro monitorování otevřených dat v reálném čase
 * @Description: 
 */

package handlers

import (
	"net/http"
	"strings"
	"os"
	"github.com/gin-gonic/gin"
	"tmc-core/src/data"
	"strconv"
)


// Funkce GetRoads vrací strom silnic pro danou oblast podle area_ref.
func GetRoads(c *gin.Context) {
	area := c.Query("area_ref")
	tmcDir := os.Getenv("TMC_TMC_DIR")
	if tmcDir == "" {
		tmcDir = "/app/static_data/tmc"
	}
	
	roads, err := data.BuildRoadList(tmcDir, area)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roads)
}




func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// Funkce RegisterLCD registruje endpoint /lcd pro hledání v bodech.
func RegisterLCD(r *gin.Engine, loader *data.Loader) {

	tsFileDir := os.Getenv("TMC_PREDEFINED_DIR")
	if tsFileDir == "" {
		tsFileDir = "/app/static_data/predefined_positions"
	}


	// Fulltextové hledání bodů.
	r.GET("/lcd", func(c *gin.Context) {
		q := strings.ToLower(c.Query("q"))

		var out []data.Point
		for _, p := range loader.Points {
			if q == "" {
				out = append(out, p)
				continue
			}
			for _, key := range []string{"ROADNAME", "FIRSTNAME", "SECONDNAME", "AREA_NAME"} {
				if strings.Contains(strings.ToLower(p[key]), q) {
					out = append(out, p)
					break
				}
			}
		}
		c.JSON(http.StatusOK, out)
	})

	// Vrácení všech bodů patřících do LCD + jeho podlokalit (podpora více lcd).
	r.GET("/lcd/admins", func(c *gin.Context) {
		// Načíst všechny lcd parametry.
		lcdParams := c.QueryArray("lcd")
		if len(lcdParams) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing lcd parameter(s)"})
			return
		}

		// Seskupit kódy a jejich podlokality.
		lcdSet := make(map[string]bool)
		for _, param := range lcdParams {
			for _, code := range strings.Split(param, ",") {
				code = strings.TrimSpace(code)
				subs := data.GetAllSubLocations(code)
				for _, sub := range subs {
					lcdSet[sub] = true
				}
			}
		}

		// Filtrovat body podle AREA_REF.
		var result []data.Point
		for _, p := range loader.Points {
			if lcdSet[p["AREA_REF"]] {
				result = append(result, p)
			}
		}
		c.JSON(http.StatusOK, result)
	})


	// Vrácení bodů pro jednu nebo více silnic/segmentů podle lcd.
    // Podpora volání: /lcd/roads?lcd=123&lcd=456 nebo /lcd/roads?lcd=123,456
    r.GET("/lcd/roads", func(c *gin.Context) {
        // načteme všechny parametry lcd (může jich být vícero)
        lcdParams := c.QueryArray("lcd")
        if len(lcdParams) == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Missing lcd parameter(s)"})
            return
        }
        // Postavíme množinu všech zadaných kódů (podpora i csv v jednom param).
        codeSet := make(map[string]bool)
        for _, param := range lcdParams {
            for _, code := range strings.Split(param, ",") {
                codeSet[strings.TrimSpace(code)] = true
            }
        }
        // Projdeme všechny body a vybereme ty, jejichž ROA_LCD nebo SEG_LCD je v množině.
        var result []data.Point
        for _, p := range loader.Points {
            if codeSet[p["ROA_LCD"]] || codeSet[p["SEG_LCD"]] {
                result = append(result, p)
            }
        }
        c.JSON(http.StatusOK, result)
    })
	
	// Vrácení bodů jak pro oblasti, tak pro silnice dle lcd.
	r.GET("/lcd/admins-roads", func(c *gin.Context) {
		adminParams := c.QueryArray("admins")
		roadParams := c.QueryArray("roads")
		if len(adminParams) == 0 && len(roadParams) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing admins or roads parameter(s)"})
			return
		}
		// Sestavení množin:
		adminSet := make(map[string]bool)
		if len(adminParams) > 0 {
			for _, param := range adminParams {
				for _, code := range strings.Split(param, ",") {
					code = strings.TrimSpace(code)
					subs := data.GetAllSubLocations(code)
					for _, sub := range subs {
						adminSet[sub] = true
					}
				}
			}
		}
		roadSet := make(map[string]bool)
		if len(roadParams) > 0 {
			for _, param := range roadParams {
				for _, code := range strings.Split(param, ",") {
					roadSet[strings.TrimSpace(code)] = true
				}
			}
		}
		// Logika filtru:
		var result []data.Point
		for _, p := range loader.Points {
			matchAdmin := len(adminParams) > 0 && adminSet[p["AREA_REF"]]
			matchRoad := len(roadParams) > 0 && (roadSet[p["ROA_LCD"]] || roadSet[p["SEG_LCD"]])
			if (len(adminParams) > 0 && len(roadParams) > 0 && matchAdmin && matchRoad) ||
				(len(adminParams) > 0 && len(roadParams) == 0 && matchAdmin) ||
				(len(adminParams) == 0 && len(roadParams) > 0 && matchRoad) {
				result = append(result, p)
			}
		}
		c.JSON(http.StatusOK, result)
	})

	r.GET("/admins", func(c *gin.Context) {
		// Přečti jeden nebo více roads parametrů i CSV.
		roadsParams := c.QueryArray("roads")
		// žádné kódy → původní strom
		if len(roadsParams) == 0 {
			tree := data.BuildAdminTree()
			c.JSON(http.StatusOK, tree)
			return
		}
		// Zavolej novou funkci, která to filtruje z loader.Points.
		filtered, err := data.BuildAdminTreeFilteredFromPoints(loader, roadsParams)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, filtered)
	})


	r.GET("/roads", func(c *gin.Context) {
		areaRef := c.Query("area_ref")
		roads, err := data.BuildRoadList("/app/static_data/tmc", areaRef)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, roads)
	})
	

	 // Vrácení TS lokalit podle LCD bodů.
	 r.GET("/predefined_positions", func(c *gin.Context) {
		// 1) Načteme lcd kódy.
		lcdParams := c.QueryArray("lcd")
		if len(lcdParams) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing lcd parameter(s)"})
			return
		}
		var lcdCodes []int
		for _, param := range lcdParams {
			for _, item := range strings.Split(param, ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				code, err := strconv.Atoi(item)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter 'lcd' must be numeric"})
					return
				}
				lcdCodes = append(lcdCodes, code)
			}
		}
	
		// 2) Načteme směry – může jich být stejně jako lcd kódů, nebo méně.
		dirParams := c.QueryArray("direction")
		var directions []string
		for _, param := range dirParams {
			for _, dir := range strings.Split(param, ",") {
				dir = strings.TrimSpace(dir)
				if dir != "positive" && dir != "negative" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Each 'direction' must be 'positive' or 'negative'"})
					return
				}
				directions = append(directions, dir)
			}
		}
	
		// 3) Sestavíme slice filtrů pro FindTSLocations
		//    Pokud pro některý LCD kód není zadán směr, použije se prázdný string (oba směry).
		var filters []data.LCDFilter
		for i, code := range lcdCodes {
			dir := ""
			if i < len(directions) {
				dir = directions[i]
			}
			filters = append(filters, data.LCDFilter{
				Code:      code,
				Direction: dir,
			})
		}
	
		// 4) Zavoláme FindTSLocations s našimi filtry a cestou k datům.
		results, err := data.FindTSLocations(filters, tsFileDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	
		// 5) Odpovíme výsledkem.
		c.JSON(http.StatusOK, results)
	})


}


