// Package mapview generates an embeddable Leaflet HTML snippet showing
// each listing as a pin colored by livability score. Uses Leaflet via CDN
// for compactness — pass --inline-map at the CLI layer if you need a fully
// offline single-file report.
package mapview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/ColeMatthewBienek/homelens/internal/redfin"
)

type pin struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Price     int     `json:"price"`
	Beds      float64 `json:"beds"`
	Baths     float64 `json:"baths"`
	SqFt      int     `json:"sqft"`
	Addr      string  `json:"addr"`
	URL       string  `json:"url"`
	Photo     string  `json:"photo,omitempty"`
	Live      int     `json:"live"`
}

const tmpl = `<div id="hl-map" style="height:480px;border-radius:12px;overflow:hidden;margin-bottom:1.5rem;border:1px solid #e2e8f0"></div>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script>
(function(){
  var pins = {{.Pins}};
  if(!pins.length){return}
  var map = L.map('hl-map');
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:18,attribution:'© OpenStreetMap'}).addTo(map);
  var bounds=[];
  function color(s){if(s>=80)return '#0d8a4a';if(s>=60)return '#5ba94c';if(s>=40)return '#c79822';if(s>=20)return '#d97534';return '#c4322b'}
  pins.forEach(function(p){
    if(!p.lat||!p.lng)return;
    var m=L.circleMarker([p.lat,p.lng],{radius:9,color:color(p.live),weight:2,fillColor:color(p.live),fillOpacity:.85}).addTo(map);
    var html='<div style="min-width:200px"><b>$'+p.price.toLocaleString()+'</b><br>'+p.addr+'<br>'+p.beds+'bd · '+p.baths+'ba · '+p.sqft+'sf<br>Livability '+p.live+'/100<br><a href="https://www.redfin.com'+p.url+'" target="_blank">View on Redfin →</a></div>';
    m.bindPopup(html);
    bounds.push([p.lat,p.lng]);
  });
  if(bounds.length){map.fitBounds(bounds,{padding:[30,30]})}
})();
</script>`

func Build(homes []redfin.Home, livability map[string]int) (template.HTML, error) {
	pins := make([]pin, 0, len(homes))
	for _, h := range homes {
		pins = append(pins, pin{
			Lat:   h.Latitude.Float(),
			Lng:   h.Longitude.Float(),
			Price: h.Price.Int(),
			Beds:  h.Beds,
			Baths: h.Baths,
			SqFt:  h.SqFt.Int(),
			Addr:  h.StreetLine.String() + ", " + h.City + ", " + h.State + " " + h.Zip,
			URL:   h.URL,
			Photo: h.PhotoURL,
			Live:  livability[h.Zip],
		})
	}
	pinJSON, err := json.Marshal(pins)
	if err != nil {
		return "", err
	}
	t, err := template.New("map").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]any{"Pins": template.JS(pinJSON)}); err != nil {
		return "", err
	}
	_ = fmt.Sprintf // keep import
	return template.HTML(buf.String()), nil
}
