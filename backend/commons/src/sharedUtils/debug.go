/**
 * @file debug.go
 * @brief Pomocné debug funkce pro sdílené použití.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

import (
	"bytes"
	"log"

	"github.com/davecgh/go-spew/spew"
)

func Dump(a ...any) {
	buffer := new(bytes.Buffer)
	spew.Fdump(buffer, a)
	log.Println(buffer.String())
}
