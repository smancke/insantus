
'use strict';

var app = angular.module('statuspage', ['ngRoute']);

app.config(function ($routeProvider, $locationProvider) {    
    $locationProvider.html5Mode(false);
    
    $routeProvider
        .when('/:env', {templateUrl: 'views/overview.html',
                        controller: 'OverviewController'})
        .otherwise({ redirectTo: '/' });    
});


app.controller('HeaderController', function($scope, $routeParams, $http) {
    $scope.environments = [];
    $scope.selectedEnv = {name: "Loading .."};

    $scope.loadEnvironments = function() {
        $http.get('/api/environments')
            .success(function(data) {
                $scope.environments = data;
                for (var i=0; i<data.length; i++) {
                    if (data[i].default) {
                        $scope.selectedEnv = data[i];
                    }
                }                
            })
            .error(function(data, status) {
                $scope.selectedEnv = {name: "Loading .. error "+ status, status: "DOWN"};
                console.log('error loading '+ status);
            });        
    }
    
    $scope.$on('$routeChangeSuccess', function(event, toState) {
        for (var i=0; i<$scope.environments.length; i++) {
            var env = $scope.environments[i];
            if (env.id == toState.pathParams.env) {
                $scope.selectedEnv = env;
            }
        }
    });

    $scope.loadEnvironments()
});

app.controller('OverviewController', function($scope,  $routeParams, $http) {
    // get check by environment
    $scope.checks = [
        {
	    name: "Loading ..",
        }
    ];

        console.log("...");
    $scope.loadChecks = function() {
        console.log("loading");
        $http.get('/api/environments/'+$routeParams.env+"/checks")
            .success(function(data) {
                $scope.checks = data;
            })
            .error(function(data, status) {
                $scope.checks = [{name: "Loading .. error "+ status, status: "DOWN"}];
                console.log('error loading checks'+ status);
            });        
    }

    $scope.loadChecks();
});

app.controller('EnvController', function($scope,  $routeParams) {

    $scope.env = $routeParams.Env;
    
});

app.run(function($rootScope) {
    $rootScope.statusBackground = function (object) {
        if (object.status == "UP") {
            return "bg-success";
        }
        if (object.status == "DOWN") {
            return "bg-danger";
        }
        if (object.status == "DEGRADED") {
            return "bg-warning";
        }
        if (object.status == "ERROR") {
            return "bg-danger";
        }
        return "bg-info";
    }
});

