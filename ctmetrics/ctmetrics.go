// Parsing of ct-metrics lgos (metrics generated by Optiq core-trading components).
package ctmetrics

// Imports
import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/frdrolland/metldr/cfg"
)

//
// Global variables
//
var (
	InputRegex *regexp.Regexp = regexp.MustCompile("(?P<timestamp>.*)\\s+\\|\\s+(.*)\\s+\\|\\s+(.*)\\s+\\|\\s+(.*)\\s+\\|\\s+(.*)\\s+\\|\\s+Connectors\\.hpp\\:\\d+\\s+\\|\\s+(?P<json>(.*))")
)

//
// Tries to parse line as if it contains "Connectors" stats
//
func ParseConnectorLines(lines []string) bool {

	n1 := InputRegex.SubexpNames()
	if nil == lines {
		return false
	}

	var bufStats []ConnectorStat = []ConnectorStat{}

	for _, s := range lines {
		r2 := InputRegex.FindStringSubmatch(s)
		if nil == r2 {
			return false
		}

		md := map[string]string{}

		for i1, n := range r2 {
			md[n1[i1]] = n
		}

		extracted := md["json"]

		if "" == extracted {
			return false
		}

		newStat := ConnectorStat{}
		if extracted != "" {
			err := json.Unmarshal([]byte(extracted), &newStat)
			if nil != err {
				fmt.Printf("ERROR while decoding JSON from file line %s - %s", extracted, err)
			}
			bufStats = append(bufStats, newStat)
		}
	}
	ProcessEvents(bufStats)

	return true
}

//
// ????????????????????
//
func GetStat(jsonstr string) (stat ConnectorStat) {
	newStat := ConnectorStat{}
	if jsonstr != "" {
		err := json.Unmarshal([]byte(jsonstr), &newStat)
		if nil != err {
			fmt.Printf("ERROR while decoding JSON from string %s - %s", jsonstr, err)
		}
	}
	//TODO Retourner une erreur plutôt
	return newStat
}

//
// Processes an event and send it to stdout or to InfluxDB, depending on which command is executed.
//
func ProcessEvent(stat ConnectorStat) error {
	var stats []ConnectorStat
	stats = []ConnectorStat{stat}
	return ProcessEvents(stats)
}

//
// Processes an event and send it to stdout or to InfluxDB, depending on which command is executed.
//
func ProcessEvents(stats []ConnectorStat) error {

	var buf bytes.Buffer
	buf = bytes.Buffer{}

	for _, newStat := range stats {
		//TODO Code à optimiser: (remplacer les fmt.Sprint par des buf.Write 'simples')
		for _, partStat := range newStat.Data.OptiqPartitions {

			for _, coreStat := range partStat.CPUCores {

				// measurement
				buf.WriteString("connector")

				// tagset
				buf.WriteString(",")
				buf.WriteString(fmt.Sprintf(`part_id=%d,part_num=%d,server_name=%s,type=%s,core=%d`, partStat.PartitionID, partStat.PartitionNumber, partStat.ServerName, partStat.InstanceType, coreStat.Core))

				// tagset
				buf.WriteString(" ")
				//buf.WriteString(fmt.Sprintf(`tz_loops_total=%d,tz_loops_used=%d,events=%d,core_usage_pct="%f",avg_events_per_loop="%f",max_events_per_loop=%d`, coreStat.TredzoneTotalLoops, coreStat.TredzoneUsedLoops, coreStat.EventsCount, coreStat.CoreUsagePercent, coreStat.AvgEventsPerLoop, coreStat.MaxEventsPerLoop))
				buf.WriteString(fmt.Sprintf(`tz_loops_total=%d,tz_loops_used=%d,events=%d,core_usage_pct=%f,avg_events_per_loop=%f,max_events_per_loop=%d`, coreStat.TredzoneTotalLoops, coreStat.TredzoneUsedLoops, coreStat.EventsCount, coreStat.CoreUsagePercent, coreStat.AvgEventsPerLoop, coreStat.MaxEventsPerLoop))

				// timestamp
				buf.WriteString(" ")
				buf.WriteString(fmt.Sprintf("%d", partStat.PublicationTime))

				// EOL
				buf.WriteString("\n")
			}
		}
	}

	switch command := cfg.Global.Command; command {
	case "import":
		// Import data ni InfluxDB
		resp, err := http.Post("http://localhost:8086/write?db=testfro", "text/plain", &buf)
		if nil != err {
			fmt.Printf("ERROR while uploading on InfluxDB: %s\n", err)
			return err
		} else {
			//TODO Faire autre chose des codes retours !!
			fmt.Printf("INFLUXDB STATUS=%s\n", resp.Status)
		}
	case "show":
		// Show only generated data on standard output
		fmt.Printf("%s", buf.String())
	default:
		log.Fatal(fmt.Sprintf("Unknown command: %s", command))
		os.Exit(10)
	}
	return nil

}
