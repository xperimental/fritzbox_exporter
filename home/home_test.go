package home

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseHomeData(t *testing.T) {
	body := `
<table class="zebra all">
<tr class="thead">
<th class="iconrow">Verbindung
</th>
<th class="name">Name
</th>
<th class="mode">Modus
</th>
<th class="temperature">Temperatur
<br>gemessen
</th>
<th class="target_temperature">Temperatur
<br>Soll
</th>
<th class="switch">Aus / An
</th>
<th class="hkrerror">
</th>
<th class="btncolumn">
</th>
</tr>
<tr>
<td class="iconrow led_green" title="Verbunden" datalabel="Wohnzimmer">
</td>
<td class="name cut_overflow">
<span title="Wohnzimmer">Wohnzimmer
</span>
</td>
<td class="mode cut_overflow">
<span>
</span>
</td>
<td class="temperature" datalabel="Temperatur gemessen">24,0 °C
</td>
<td class="target_temperature" datalabel="Temperatur Soll">
<span class="numinput ">
<button id="uiNumDown:TSOLL_16" class="svgbtn" type="button">
<svg viewbox="0 0 16 16" height="1.3em" width="1.3em">
<rect y="1" x="0" rx="2" height="14" width="16" fill="#9fb0bc">
</rect>
<path stroke-width="1" stroke="#ffffff" d="M 4,8 L 12,8">
</path>
</svg>
</button>
<span class="numdisplay">
<span id="uiNumDisplay:TSOLL_16">22,5
</span>
<span id="uiNumUnit:TSOLL_16"> °C
</span>
</span>
<button id="uiNumUp:TSOLL_16" class="svgbtn" type="button">
<svg viewbox="0 0 16 16" height="1.3em" width="1.3em">
<rect y="1" x="0" rx="2" height="14" width="16" fill="#9fb0bc">
</rect>
<path stroke-width="1" stroke="#ffffff" d="M 4,8 L 12,8 M 8,4 L 8,12">
</path>
</svg>
</button>
<input id="uiNum:TSOLL_16" type="hidden" name="TSOLL_16" value="22.5">
</span>
</td>
<td class="switch">
<div id="uiSwitch:16">
</div>
</td>
<td class="hkrerror">
</td>
<td datalabel="" class="btncolumn">
<button type="submit" name="edit" value="16" class="icon edit" title="Bearbeiten">
</button>
<button onclick="return confirmDelete(&quot;Möchten Sie den Heizkörperregler &#92;&quot;Wohnzimmer&#92;&quot; von der FRITZ!Box abmelden?&quot;);" type="submit" name="delete" value="16" class="icon delete" title="Abmelden">
</button>
</td>
</tr>
</table>
<div class="btn_form">
<button id="ui_NewGroup" type="submit" name="new_group">Neue Gruppe
</button>
<button name="new_device" type="submit">Neues Gerät anmelden
</button>
<br>
</div>`
	r := strings.NewReader(body)
	home := homeData{}
	err := parseHomeData(&home, r)
	if err != nil {
		t.Errorf("got error '%s'", err)
	}

	expected := homeData{
		Thermostats: []thermostat{
			{
				Name:               "Wohnzimmer",
				CurrentTemperature: 24.0,
				TargetTemperature:  22.5,
			},
		},
	}
	if !reflect.DeepEqual(home, expected) {
		t.Errorf("got %#v, wanted %#v", home, expected)
	}
}
