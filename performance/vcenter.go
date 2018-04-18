package performance

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type Metric struct {
	Time  int64   `json:"x"`
	Value float32 `json:"y"`
}

type Series struct {
	Name     string   `json:"name"`
	Instance string   `json:"instance"`
	Metric   string   `json:"metric"`
	Entity   string   `json:"entity"`
	Unit     string   `json:"unit"`
	Data     []Metric `json:"data"`
}

func getPerfManager(ctx context.Context, uri string) (*performance.Manager, error) {

	u, _ := url.Parse(uri)

	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return nil, err
	}

	return performance.NewManager(client.Client), nil
}

func getFinder(ctx context.Context, perfManager *performance.Manager, datacenter string) (*find.Finder, error) {

	finder := find.NewFinder(perfManager.Client(), false)

	dc, err := finder.Datacenter(ctx, datacenter)

	if err != nil {
		return nil, err
	}
	finder.SetDatacenter(dc)

	return finder, nil
}

func doQuery(ctx context.Context, perfManager *performance.Manager, metrics []string, refs []types.ManagedObjectReference) ([]*Series, error) {

	spec := types.PerfQuerySpec{
		Format:     string(types.PerfFormatNormal),
		MaxSample:  int32(180),
		MetricId:   []types.PerfMetricId{{Instance: "*"}},
		IntervalId: int32(20),
	}

	sample, err := perfManager.SampleByName(ctx, spec, metrics, refs)
	if err != nil {
		return nil, err
	}

	result, err := perfManager.ToMetricSeries(ctx, sample)
	if err != nil {
		return nil, err
	}

	counters, err := perfManager.CounterInfoByName(ctx)
	if err != nil {
		return nil, err
	}

	results, err := metricArray(perfManager, result, counters, os.Stdout)

	return results, nil
}

func FindHosts(vcenterURI, datacenter string) ([]string, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	perfManager, err := getPerfManager(ctx, vcenterURI)
	if err != nil {
		return nil, err
	}

	//var refs []types.ManagedObjectReference

	finder, err := getFinder(ctx, perfManager, datacenter)
	if err != nil {
		return nil, err
	}

	var ips []string

	hosts, err := finder.HostSystemList(ctx, "*")
	if err != nil {
		log.Errorf("Hosts errors: %s", err)
		return nil, err
	}

	for _, host := range hosts {
		hostIPs, err := host.ManagementIPs(ctx)
		if err != nil {
			return nil, err
		}

		for _, ip := range hostIPs {
			ips = append(ips, ip.String())
		}
	}

	return ips, nil
}

func VMs(vcenterURI string, datacenter string, vms []string, metrics []string) ([]*Series, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	perfManager, err := getPerfManager(ctx, vcenterURI)
	if err != nil {
		return nil, err
	}

	var refs []types.ManagedObjectReference

	finder, err := getFinder(ctx, perfManager, datacenter)
	if err != nil {
		return nil, err
	}

	for _, vmPath := range vms {
		vm, err := finder.VirtualMachine(ctx, vmPath)

		if err != nil {
			return nil, err
		}
		refs = append(refs, vm.Reference())
	}

	return doQuery(ctx, perfManager, metrics, refs)
}

func Hosts(vcenterURI string, datacenter string, hostnames []string, metrics []string) ([]*Series, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	perfManager, err := getPerfManager(ctx, vcenterURI)
	if err != nil {
		return nil, err
	}

	var refs []types.ManagedObjectReference

	finder, err := getFinder(ctx, perfManager, datacenter)
	if err != nil {
		return nil, err
	}

	for _, hostname := range hostnames {
		host, err := finder.HostSystem(ctx, hostname)
		if err != nil {
			return nil, err
		}
		refs = append(refs, host.Reference())
	}

	return doQuery(ctx, perfManager, metrics, refs)
}

func metricArray(p *performance.Manager, sample []performance.EntityMetric, counters map[string]*types.PerfCounterInfo, w io.Writer) ([]*Series, error) {
	var seriesResults []*Series

	for i := range sample {
		metric := sample[i]

		name := EntityName(p, metric.Entity)

		for _, value := range metric.Value {

			var seriesName string
			if value.Instance != "" {
				seriesName = fmt.Sprintf("%s - %s", value.Name, value.Instance)
			} else {
				seriesName = value.Name
			}

			series := Series{
				Name:     seriesName,
				Instance: value.Instance,
				//	Unit:     value.Unit,
				Metric: value.Name,
				Entity: name,
			}
			for j := range value.Value {
				floatValue, _ := strconv.ParseFloat(value.Format(value.Value[j]), 32)

				series.Data = append(series.Data, Metric{
					Time:  metric.SampleInfo[j].Timestamp.Unix(),
					Value: float32(floatValue),
				})
			}

			seriesResults = append(seriesResults, &series)
		}
	}
	return seriesResults, nil
}

func EntityName(p *performance.Manager, e types.ManagedObjectReference) string {
	var me mo.ManagedEntity
	_ = p.Properties(context.Background(), e, []string{"name"}, &me)

	name := me.Name

	return name
}

func sampleInfoTimes(m *performance.EntityMetric) []string {
	vals := make([]string, len(m.SampleInfo))

	for i := range m.SampleInfo {
		vals[i] = m.SampleInfo[i].Timestamp.Format(time.RFC3339)
	}

	return vals
}
