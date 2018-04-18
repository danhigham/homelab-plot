$(document).ready(function() {

  var dc="DC0"
  var graphs = []
  var config = {}
  var metricsToPlot = []

  async.waterfall([
    function(callback) {
      $.getJSON("/js/config.json", function(data) {
        config=data;
        metricsToPlot = _.map(config.chartMetrics, function(m) { return m.key; });
        callback(null, config);
      });
    },
    function(config, callback) {
      doPost("/findhosts.json", { "dc": dc }, function (hosts) {
        pollHosts(dc, hosts, renderHosts);
        setInterval(function(){ pollHosts(dc, hosts, renderHosts); }, 20000);
        callback(null, config, hosts);
      });
    },
    function(config, hosts, callback) {
      $.getJSON("/deployments.json", null, function(data) {
        callback(null, config, hosts, data);
      });
    },
    function(config, hosts, deployments, callback) {
      _.each (deployments, function (deployment) {
        var url = "/deployment/" + deployment.name + "/vms.json";

        $.getJSON(url, null, function(vms) {
          pollVMs(dc, vms, deployment, renderVMs);
          setInterval(function() { pollVMs(dc, vms, deployment, renderVMs); }, 20000);
        });

      });
    },
  ], function (err, result) {
    console.log(result);
  });

  function renderHosts(hostMetrics) {

    uniqueHosts = _.uniq(_.map(hostMetrics, function(metric) { return metric.entity } ));
    uniqueMetrics = _.uniq(_.map(hostMetrics, function(metric) { return metric.metric } ));

    _.each (uniqueHosts, function (host) {
      if ($('ul#hosts > li#host-' + host.replace(/\./g, "-")).length == 0) {
        var li = renderTemplate("host_tmpl", { "name" : host, "id": host.replace(/\./g, "-") });
        $('ul#hosts').append(li);
      }
      var chartsList = $("ul#hosts > li#host-" + host.replace(/\./g, "-") + " > div.charts > ul");
      _.each (uniqueMetrics, function(metric) {
        var series = _.filter(hostMetrics, function(hm) { return ((hm.metric == metric) && (hm.entity == host)); });

        var graph = _.find(graphs, function(g) { return ((g.entity == host) && (g.metric == metric)); });
        if (graph) {
          _.each(graph.graph.series, function(oldSeries) {
            s = _.find(series, function(newSeries) { return (newSeries.instance == oldSeries.instance); });
            oldSeries.data = s.data;
          });
          graph.graph.update();
        } else {

          var li=document.createElement("li");
          $(li).append("<div class=\"y-axis\"></div><div class=\"chart-body\"></div>");
          chartsList.append(li);

          var metrics = _.filter(hostMetrics, function(hm) { return ((hm.metric == metric) && (hm.entity == host)); });
          var axisTarget = $(li).find("div.y-axis");
          var chartTarget = $(li).find("div.chart-body");
          var graph = renderChart(metrics, chartTarget, axisTarget);
          graphs.push({entity: host, metric: metric, graph: graph});
        }
      });
    });
  }

  function renderVMs(vmMetrics) {

    _.each(_.sortBy(vmMetrics, function(metric) { return metric.job + "" + metric.job_index + "" + metric.metric; }), function(metric) {
      // does the deployment list exist
      if ($("#deployment-" + metric.deployment).length == 0) {
        // render deployment
        var li = renderTemplate("deployment_tmpl", { "name" : metric.deployment, "id": metric.deployment });
        $('ul#deployments').append(li);
      }
      deploymentVMList = $("ul#deployments > li#deployment-" + metric.deployment + " > div.vms > ul");

      if (deploymentVMList.find("li#" + metric.entity).length == 0) {
        var li = renderTemplate("vm_tmpl", { "name" : metric.job, "index": metric.job_index, "id": metric.entity });
        deploymentVMList.append(li);
      }
      var chartsList = deploymentVMList.find("li#" + metric.entity + " > div.charts > ul");

      var graph = _.find(graphs, function(g) { return ((g.entity == metric.entity) && (g.metric == metric.metric)); });
      if (graph) {
        graph.graph.series[0].data = metric.data;
        graph.graph.update();
      } else {
        var li=document.createElement("li");
        $(li).append("<div class=\"y-axis\"></div><div class=\"chart-body\"></div>");
        chartsList.append(li);

        var axisTarget = $(li).find("div.y-axis");
        var chartTarget = $(li).find("div.chart-body");
        var graph = renderChart([metric], chartTarget, axisTarget);
        graphs.push({entity: metric.entity, metric: metric.metric, graph: graph});
      }
    });
  }

  function renderChart(metrics, chartTarget, axisTarget) {

    var palette = new Rickshaw.Color.Palette( { scheme: 'classic9' } );
    _.each(metrics, function(m) {
      m.color = palette.color();
    });

    var ticksTreatment = "glow";
    var graph = new Rickshaw.Graph({
        element: chartTarget[0],
        width: 400,
        height: 150,
        renderer: 'line',
        series: metrics})

    var metricConfig = _.find(config.chartMetrics, function(m) { return m.key == metrics[0].metric; });

    if (metricConfig.max) { graph.max = metricConfig.max; }
    if (metricConfig.min) { graph.min = metricConfig.min; }

    graph.render();

    var detail = new Rickshaw.Graph.HoverDetail({
        graph: graph,
        xFormatter: function(x) { return new Date(x * 1000).toLocaleTimeString(); }
    });
    var xAxis = new Rickshaw.Graph.Axis.Time({
        graph: graph,
        ticksTreatment: ticksTreatment,
        timeFixture: new Rickshaw.Fixtures.Time.Local()
    });
    xAxis.render();
    var yAxis = new Rickshaw.Graph.Axis.Y( {
        element: axisTarget[0],
        orientation: 'left',
        graph: graph,
        tickFormat: Rickshaw.Fixtures.Number.formatKMBT,
        ticksTreatment: ticksTreatment
    } );
    yAxis.render();

    return graph;
  }

  function renderTemplate(name, ctx) {
    var template = $('#' + name).html();
    Mustache.parse(template);   // optional, speeds up future uses
    return Mustache.render(template, ctx);
  }

  function pollHosts(dc, hosts, callback) {
    doPost("/hosts.json", { "dc": dc, "hosts": hosts, "metrics": metricsToPlot }, function (hostMetrics) {
      callback(hostMetrics);
    });
  }

  function pollVMs(dc, vms, deployment, callback) {
    var vmIDs = _.map(vms, function(vm) { return vm.vm_cid; });

    doPost("/vms.json", { "dc": dc, "vms": vmIDs, "metrics": metricsToPlot }, function (vmMetrics) {
      // render vms
      _.each(vmMetrics, function(metric) {
        metric.deployment = deployment.name;
        var vm = _.find(vms, function(vm) { return vm.vm_cid == metric.entity; });
        metric.job = vm.job_name;
        metric.job_index = vm.index;
      });

      callback(vmMetrics);
    });
  }

  function doPost(url, data, callback) {
    $.ajax({
      type: "POST",
      data: JSON.stringify(data),
      contentType: "application/json",
      url: url,
    }).done(function(data) {
      callback(data);
    });
  }
});
