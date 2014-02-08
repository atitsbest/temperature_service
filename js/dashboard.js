(function() {
  var colors = [
       '#A6C776',
       '#82578F',
       '#D98C80',
       '#D8B380',
       '#56748B'];

  function generate_options() {
    return {
      series: [],
      chart: {
          renderTo: 'mainChart',
          type: 'area',
          height: 200,
	  spacingLeft: 0,
	  spacingRight: 0,
	  backgroundColor: '#333'
      },
      colors: colors,
      title: {
          text: ''
      },
      subtitle: {
          text: ''
      },
      xAxis: {
          type: 'datetime',
	  lineWidth: 3,
	  lineColor: '#666',
	  tickColor: '#666',
          dateTimeLabelFormats: { // don't display the dummy year
              month: '%e. %b',
              year: '%b'
          },
	  labels: {
	  	style: {
			color: '#666',
			"font-style": 'italic'
		}
	  }
      },
      yAxis: {
          title: {
              text: null
          },
	  labels: {
	  	style: {
			color: '#666',
			"font-style": 'italic'
		}
	  },
	  gridLineColor: '#666'
      },
      legend: {
          enabled: false
      },
      plotOptions: {
          area: {
            //fillColor: null,
            fillOpacity: 0.3,
            marker: {
              enabled: false
            }
          },
      },
      tooltip: {
          enabled: true,
          formatter: function() {
                  return '<b>'+ this.series.name +'</b><br/>'+
                  Highcharts.dateFormat('%e. %b', this.x) +': '+ this.y +' °C';
          }
      },
      credits: {
        enabled: false
      }
    };
  }

  function create_serie_for(data, name)
  {
      var values = data[name]
        .map(function(v) { return [v.d*1000, v.v/100.0]; });

      return {
        name: name,
        data: values
      };
  }

  angular.module('temperature', []).

    config(function($interpolateProvider) {
      $interpolateProvider.startSymbol('%%'); // Damit wir keine Probleme mit den Go-Templates bekommen.
      $interpolateProvider.endSymbol('%%');
    }).

    controller('DashboardCtrl', function($scope, $http) {
      // {"sensor1_name": [{"d": 1234567, "v": 2530}, ...], "sensor2_name": [...]}
      $scope.measurements = {};
      // Liste der Sensoren.
      $scope.sensors = [];
      $scope.colors = colors;

      // Liefert eine Liste aller Sensoren.
      $scope.$watch("measurements", function() {
        $scope.sensors =  _($scope.measurements).keys();
      }, true);

      /**
       *  Letzte Temperatur des Sensors
       *  @param sensor string
       */
      $scope.currentTemperature = function(sensor) {
        var data = $scope.measurements[sensor];
        return _(data).last().v / 100.0;
      };

      $http.get('api/measurements.json').
        success(function(data) {
          $scope.measurements = data;
          _(_(data).keys()).each(function(s) {
            chart.addSeries( create_serie_for(data, s) );
          });
        }).
        error(alert)
        ['finally'](function() { chart.hideLoading(); });

      $scope.chartConfig = generate_options();

      // Chart gleich mal anzeigen, auch wenn wir noch keine Daten haben...
      var chart = new Highcharts.Chart($scope.chartConfig);
      // ...dafür zeigen wir eine "laden..."-Meldung.
      chart.showLoading();

    });

})();
