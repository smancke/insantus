
'use strict';

var app = angular.module('statuspage', ['ngRoute']);

app.config(function ($routeProvider, $locationProvider) {    
    $locationProvider.html5Mode(false);
    
    $routeProvider
        .when('/', {templateUrl: 'views/overview.html',
                    controller: 'OverviewController'})
        .when('/:env', {templateUrl: 'views/env.html',
                        controller: 'EnvController'})
        .otherwise({ redirectTo: '/' });
});


app.factory('store', function($rootScope, $http, $interval) {
    var store = {};
    store.data = {};
    
    store.data.environments = [];
    store.data.selectedEnv = undefined;
    store.data.checks = [{name: "Loading .."}];
    store.data.sinceLastCheckUpdate = undefined;
    store.checkUpdateTimestamp = undefined;
    store.selectedEnvId = undefined;
    store.stopTimer = undefined;

    store._loadEnvironments = function() {
        $http.get('/api/environments')
            .success(function(data) {
                data.sort(compareByName);                
                store.data.environments = data;
                if (angular.isDefined(store.selectedEnvId)) {
                    store.selectEnv(store.selectedEnvId);
                }                               
            })
            .error(function(data, status) {
                store.data.selectedEnv = {name: "Loading .. error "+ status, status: "DOWN"};
                console.log('error loading '+ status);
            });        
    }

    store._loadChecks = function() {
        $http.get('/api/environments/'+store.selectedEnvId+"/checks")
            .success(function(data) {
                data.sort(compareByName);
                store.data.checks = data;
                store.checkUpdateTimestamp = Date.now();
                store.data.sinceLastCheckUpdate = 0;
            })
            .error(function(data, status) {
                store.data.checks = [{name: "Loading .. error "+ status, status: "DOWN"}];
                console.log('error loading checks'+ status);
            });        
    }

    store.selectEnv = function(envId) {
        store.selectedEnvId = envId;
        for (var i=0; i<store.data.environments.length; i++) {
            var env = store.data.environments[i];
            if (env.id == envId) {
                store.data.selectedEnv = env
            }
        }
        store._loadChecks();
    }
    
    store.stopReloading = function() {
        if (angular.isDefined(store.stopTimer)) {
            $interval.cancel(store.stopTimer);
        }
    }
    
    store.reload = function() {
        store._loadEnvironments();
        if (angular.isDefined(store.selectedEnvId)) {
            store.selectEnv(store.selectedEnvId);
        }
    }
    store.enableReloading = function() {
        store.reload();
        store.stopTimer = $interval(function(){
            store.reload();
        }, 10000);
    }

    $interval(function(){
        store.data.sinceLastCheckUpdate = Date.now() - store.checkUpdateTimestamp;
    }, 1000);
    
    store.enableReloading();
    return store;
});

app.controller('HeaderController', function($scope, $routeParams, $http, store) {
    $scope.store = store.data
});
           
app.controller('OverviewController', function($scope, $routeParams, $http, store) {
    $scope.store = store.data
});

app.controller('EnvController', function($scope,  $routeParams, $http, store) {
    $scope.store = store.data
    store.selectEnv($routeParams.env);    
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

    $rootScope.formatSince = function(millis) {
        return Math.round(millis/1000) + 's';
    }
});

function compareByName(a,b) {
    if(a.name < b.name) return -1;
    if(a.name > b.name) return 1;
    return 0;
}
