/**
 * @file ternaryOperator.go
 * @brief Pomocná funkce napodobující ternární operátor.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

func Ternary[T any](cond bool, r1 T, r2 T) T {
	if cond {
		return r1
	} else {
		return r2
	}
}
