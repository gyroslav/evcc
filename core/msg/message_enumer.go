// Code generated by "enumer -type=Message"; DO NOT EDIT.

//
package msg

import (
	"fmt"
)

const _MessageName = "ErrorWarnBatteryConfiguredBatteryPowerBatterySoCGridConfiguredGridCurrentsGridPowerPrioritySoCPvConfiguredPvPowerSiteTitleActivePhasesChargeConfiguredChargeCurrentChargeCurrentsChargedEnergyChargeDurationChargeEstimateChargePowerChargeRemainingEnergyChargingClimaterConnectedConnectedDurationEnabledHasVehicleMaxCurrentMinCurrentMinSoCModePhasesRangeRemoteDisabledRemoteDisabledSourceSocCapacitySocChargeSocLevelsSocTitleTargetSoCTargetTimeTimerActiveTimerSetTitle"

var _MessageIndex = [...]uint16{0, 5, 9, 26, 38, 48, 62, 74, 83, 94, 106, 113, 122, 134, 150, 163, 177, 190, 204, 218, 229, 250, 258, 266, 275, 292, 299, 309, 319, 329, 335, 339, 345, 350, 364, 384, 395, 404, 413, 421, 430, 440, 451, 459, 464}

func (i Message) String() string {
	i -= 1
	if i < 0 || i >= Message(len(_MessageIndex)-1) {
		return fmt.Sprintf("Message(%d)", i+1)
	}
	return _MessageName[_MessageIndex[i]:_MessageIndex[i+1]]
}

var _MessageValues = []Message{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44}

var _MessageNameToValueMap = map[string]Message{
	_MessageName[0:5]:     1,
	_MessageName[5:9]:     2,
	_MessageName[9:26]:    3,
	_MessageName[26:38]:   4,
	_MessageName[38:48]:   5,
	_MessageName[48:62]:   6,
	_MessageName[62:74]:   7,
	_MessageName[74:83]:   8,
	_MessageName[83:94]:   9,
	_MessageName[94:106]:  10,
	_MessageName[106:113]: 11,
	_MessageName[113:122]: 12,
	_MessageName[122:134]: 13,
	_MessageName[134:150]: 14,
	_MessageName[150:163]: 15,
	_MessageName[163:177]: 16,
	_MessageName[177:190]: 17,
	_MessageName[190:204]: 18,
	_MessageName[204:218]: 19,
	_MessageName[218:229]: 20,
	_MessageName[229:250]: 21,
	_MessageName[250:258]: 22,
	_MessageName[258:266]: 23,
	_MessageName[266:275]: 24,
	_MessageName[275:292]: 25,
	_MessageName[292:299]: 26,
	_MessageName[299:309]: 27,
	_MessageName[309:319]: 28,
	_MessageName[319:329]: 29,
	_MessageName[329:335]: 30,
	_MessageName[335:339]: 31,
	_MessageName[339:345]: 32,
	_MessageName[345:350]: 33,
	_MessageName[350:364]: 34,
	_MessageName[364:384]: 35,
	_MessageName[384:395]: 36,
	_MessageName[395:404]: 37,
	_MessageName[404:413]: 38,
	_MessageName[413:421]: 39,
	_MessageName[421:430]: 40,
	_MessageName[430:440]: 41,
	_MessageName[440:451]: 42,
	_MessageName[451:459]: 43,
	_MessageName[459:464]: 44,
}

// MessageString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func MessageString(s string) (Message, error) {
	if val, ok := _MessageNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to Message values", s)
}

// MessageValues returns all values of the enum
func MessageValues() []Message {
	return _MessageValues
}

// IsAMessage returns "true" if the value is listed in the enum definition. "false" otherwise
func (i Message) IsAMessage() bool {
	for _, v := range _MessageValues {
		if i == v {
			return true
		}
	}
	return false
}
