/**
 * @file sync.go
 * @brief Pomocné synchronizační funkce.
 *
 * @author Vojtěch Hubáček
 * @ingroup riot_preprocessors_commons
 *
 * @par Autorský podíl
 * - Vojtěch Hubáček: převzetí, údržba a doplnění sdílených kontraktů potřebných pro preprocesory v rámci samostatného repozitáře.
 */
package sharedUtils

import "sync"

func WaitForAll(functions ...func()) {
	wg := new(sync.WaitGroup)
	wg.Add(len(functions))
	for _, function := range functions {
		go func(function func()) {
			defer wg.Done()
			function()
		}(function)
	}
	wg.Wait()
}
