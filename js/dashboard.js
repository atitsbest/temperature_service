(function() {

  var colors= [
   ['#A6C776', '#A0C170', '#fff'],
   ['#82578F', '#765189', '#fff'], 
   ['#D98C80', '#D38674', '#fff'], 
   ['#D8B380', '#D2AD7A', '#fff'],
   ['#56748B', '#527078', '#fff']
  ];

  function generate_options(colors) {
    return {
      series: [],
      chart: {
          type: 'area',
          height: 200
      },
      title: {
          text: ''
      },
      subtitle: {
          text: ''
      },
      xAxis: {
          type: 'datetime',
          dateTimeLabelFormats: { // don't display the dummy year
              month: '%e. %b',
              year: '%b'
          }
      },
      yAxis: {
          title: {
              text: 'Temperatur'
          }
      },
      legend: {
          enabled: false
      },
      plotOptions: {
          series: {
            //fillColor: null,
            fillOpacity: 0.2
          },
          marker: {
            enabled: false
          }
      },
      tooltip: {
          enabled: true,
          formatter: function() {
                  return '<b>'+ this.series.name +'</b><br/>'+
                  Highcharts.dateFormat('%e. %b', this.x) +': '+ this.y +' Â°C';
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

  angular.module('temperature', ['highcharts-ng']).
    
    config(function($interpolateProvider) {
      $interpolateProvider.startSymbol('%%');
      $interpolateProvider.endSymbol('%%');
    }).

    controller('DashboardCtrl', function($scope, $http) {
      // {"sensor1_name": [{"d": 1234567, "v": 2530}, ...], "sensor2_name": [...]}
      $scope.measurements = {};
      // Liste der Sensoren.
      $scope.sensors = [];

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

      // Messungen für alle Sensoren laden.
      $http.get('api/measurements.json').
        success(function(data) {
          $scope.measurements = data;
          _(_(data).keys()).each(function(s) {
            $scope.chartConfig.series.push( create_serie_for(data, s) );
          });
        }).
        error(alert);

      $scope.chartConfig = generate_options(colors[0]);
    });


  // $(function() {
  //   $.when($.getJSON("/api/measurements.json")).
  //     then(function(data) {
  //       $(".sensor").each(function(i, panel) {
  //
  //         var sensor = $(panel).data('sensor');
  //         if (sensor !== undefined) {
  //           // Farbe fÃ¼r diesen Chart.
  //           var current_colors = colors[i%colors.length];
  //         
  //           // Daten fÃ¼r den einen Sensor filtern.
  //           var serie = create_serie_for(data, sensor);
  //
  //           var options = _.extend(
  //             generate_options(current_colors),
  //             { series: [serie] });
  //
  //           // Chart in der DOM platzieren.
  //           $(panel).find('.chart')
  //             .highcharts(options);
  //
  //           // TODO: Hintegrundfarbe setzten besser machen!
  //           $(panel).css('background', current_colors[0]);
  //         }
  //
  //       });
  //
  //     }).
  //     fail(function(error) {
  //       alert(error);
  //     });
  //
  //
  //     // SSE fÃ¼r TemperaturÃ¤nderungen init.
  //     var source = new EventSource('/realtime/measurements');
  //       source.addEventListener('update', function(e) {
  //       update = JSON.parse(e.data);
  //       temp = update.data.v / 100.0;
  //       ago = $.timeago(new Date(update.data.d*1000));
  //       $sensor = $('[data-sensor="' + update.sensor + '"]');
  //       $sensor.find('.temperature > span').text(temp);
  //       $sensor.find('.timeago > .val').text(ago);
  //     });
  //     
  // });

})();
