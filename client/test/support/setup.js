var sinon = require('sinon')

var Immutable = require('immutable')
Immutable.Iterable.noLengthWarning = true

var Reflux = require('reflux')
Reflux.nextTick(callback => window.setTimeout(callback, 0))

var support = {}

support.setupClock = function() {
  var clock = sinon.useFakeTimers()

  // manually fix Sinon #624 until it updates Lolex to 1.2.0
  Date.now = function() { return Date().getTime() }

  // set up fake clock to work with lodash
  var _ = require('lodash')

  var origDebounce = _.debounce
  var origThrottle = _.throttle

  var mock_ = _.runInContext(window)
  _.debounce = mock_.debounce
  _.throttle = mock_.throttle

  var origRestore = clock.restore.bind(clock)
  clock.restore = function() {
    _.debounce = origDebounce
    _.throttle = origThrottle
    origRestore()
  }

  // remove erroneous entry from coverage listing
  Date.now()

  return clock
}

support.listenOnce = function(listenable, callback) {
  var remove = listenable.listen(function() {
    remove()
    callback.apply(this, arguments)
  })
}

support.resetStore = function(store) {
  store.init()
  store.emitter.removeAllListeners()
}

window.Heim = {
  setFavicon: function() {},
}

module.exports = support
