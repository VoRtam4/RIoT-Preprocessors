/**
 * @file generators.go
 * @brief Pomocné generátory identifikátorů a technických hodnot.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

import "sync/atomic"

func SequentialNumberGenerator() func() uint32 {
	counter := uint32(0)
	return func() uint32 {
		return atomic.AddUint32(&counter, 1)
	}
}
