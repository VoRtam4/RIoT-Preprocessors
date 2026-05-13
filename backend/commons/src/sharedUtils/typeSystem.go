/**
 * @file typeSystem.go
 * @brief Pomocné funkce pro práci s typy a hodnotami.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

func TypeIs[T any](subject any) bool {
	_, ok := subject.(T)
	return ok
}
