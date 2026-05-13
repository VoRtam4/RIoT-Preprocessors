/**
 * @file errorHandling.go
 * @brief Pomocné funkce pro sjednocené ošetření chyb.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

import "log"

// TerminateOnError is a helper function that logs the supplied error message along some other information and then terminates the program in case of error.
func TerminateOnError(err error, errorMessage string) {
	if err != nil {
		log.Fatalf("%s: %s\n", errorMessage, err.Error())
	}
}

// LogPossibleErrorThenProceed is a helper function that logs the supplied error message along some other information in case of error. It does not terminate the program.
func LogPossibleErrorThenProceed(err error, errorMessage string) {
	if err != nil {
		log.Printf("%s: %s\n", errorMessage, err.Error())
	}
}
