/**
 * @file slice.go
 * @brief Pomocné funkce pro práci se slices.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

func SliceOf[T any](items ...T) []T {
	return items
}

func EmptySlice[T any]() []T {
	return make([]T, 0)
}
