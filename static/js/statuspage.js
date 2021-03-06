
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
    store.data.downtimes = [];
    store.data.sinceLastCheckUpdate = undefined;
    store.checkUpdateTimestamp = undefined;
    store.selectedEnvId = undefined;
    store.stopTimer = undefined;
    store.reloadingEnabled = undefined;

    store.openedChecks = [];

    store._loadEnvironments = function() {
        $http.get('/api/environments')
            .success(function(data) {
                store.data.environments = []
                for (var key in data) {
                    if (key != "status") {
                        store.data.environments.push(data[key]);
                    }
                }
                store.data.environments.sort(compareByName);                
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
        var envId = store.selectedEnvId;

        $http.get('/api/environments/'+envId)
            .success(function(data) {
                if (envId != store.selectedEnvId) {
                    // only update, if the eventid has not changed in the meantime
                    return;
                }
                
                store.checkUpdateTimestamp = Date.now();
                store.data.downtimes = data.downtimes;
                store.data.sinceLastCheckUpdate = 0;
                data.checks.sort(compareByName);
                store.data.checks = data.checks;
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
        store.reloadingEnabled = false
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
        store.reloadingEnabled = true
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


app.run(function($rootScope, store) {
    $rootScope.statusBackground = function (object) {
        var s = object.status;
        if ('statusTo' in object) {
            s = object.statusTo;
        }
        
        if (s == "UP") {
            return "bg-success";
        }
        if (s == "DOWN") {
            return "bg-danger";
        }
        if (s == "DEGRADED") {
            return "bg-warning";
        }
        if (s == "ERROR") {
            return "bg-danger";
        }
        return "bg-info";
    }

    $rootScope.toggleDetails = function (check) {
        var openedChecks = store.openedChecks;
        var index = openedChecks.indexOf(check);

        if (index === -1) {
            openedChecks.push(check);
        } else {
            openedChecks.splice(index, 1);
        }
    }

    $rootScope.isInOpenedChecks = function (check) {
        var openedChecks = store.openedChecks;
        return openedChecks.indexOf(check) !== -1;
    }

    var oneMinute = 1000 * 60
    var oneHour = 1000 * 60 * 60
    var oneDay = 24 * oneHour    
    $rootScope.sinceString = function(start, end) {
        start = Date.parse(start)
        end = Date.parse(end)
        if (end < start) {
            end = new Date()
        }

        var duration = new Date(end - start)
        if (duration > oneDay) {
            return Math.round(duration / oneDay ) + "d"
        }
        if (duration > oneHour) {
            return Math.round(duration / oneHour ) + "h"
        }
        if (duration > oneMinute) {
            return Math.round(duration / oneMinute ) + "m"
        }
        return Math.round(duration/1000) +"s"
    }

    $rootScope.formatSince = function(millis) {
        return Math.round(millis/1000) + 's';
    }

    $rootScope.isoToDate = function(isoDateString) {
        return Date.parse(isoDateString)
    }
});

function compareByName(a,b) {
    if(a.name < b.name) return -1;
    if(a.name > b.name) return 1;
    return 0;
}
