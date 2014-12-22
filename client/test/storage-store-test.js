var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')


describe('storage store', function() {
  var storage = require('../lib/stores/storage')
  var getItem = localStorage.getItem
  var setItem = localStorage.setItem
  var fakeStorage

  beforeEach(function() {
    fakeStorage = {}
    sinon.stub(localStorage, 'getItem', function(key) {
      return fakeStorage[key]
    })
    sinon.stub(localStorage, 'setItem', function(key, value) {
      fakeStorage[key] = value
    })
    support.resetStore(storage.store)
  })

  afterEach(function() {
    // stub.restore() seems to fail here.
    localStorage.getItem = getItem
    localStorage.setItem = setItem
  })

  it('should load JSON from localStorage upon init', function() {
    var data = {it: 'works'}
    fakeStorage.data = JSON.stringify(data)
    storage.store.init()
    assert.deepEqual(storage.store.getInitialState(), data)
  })

  describe('set action', function() {
    var testKey = 'testKey'
    var testValue = {test: true}

    it('should save JSON to localStorage', function() {
      storage.store.set(testKey, testValue)
      support.clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'testKey': testValue
      }))
    })

    it('should trigger an update event', function(done) {
      support.listenOnce(storage.store, function(state) {
        assert.equal(state[testKey], testValue)
        done()
      })

      storage.store.set(testKey, testValue)
    })
  })
})