var _ = require('lodash')
var Reflux = require('reflux')


var actions = Reflux.createActions([
  'enable',
  'disable',
])
_.extend(module.exports, actions)

actions.enable.sync = true

module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    {chatUpdate: require('./chat').store},
  ],

  init: function() {
    this.state = {
      enabled: null,
      supported: 'Notification' in window
    }

    this.focus = true
    this.notification = null
    this.notificationTimeout = null

    if (this.state.supported) {
      this.state.permission = Notification.permission == 'granted'
    }

    window.addEventListener('focus', this.onFocus.bind(this), false)
    window.addEventListener('blur', this.onBlur.bind(this), false)
  },

  onFocus: function() {
    this.focus = true
    this.closeNotification()
  },

  onBlur: function() {
    this.focus = false
  },

  getDefaultData: function() {
    return this.state
  },

  enable: function() {
    if (this.state.permission) {
      this.state.enabled = true
      this.trigger(this.state)
    } else {
      Notification.requestPermission(this.onPermission)
    }
  },

  disable: function() {
    this.state.enabled = false
    this.trigger(this.state)
  },

  onPermission: function(permission) {
    if (permission === "granted") {
      this.state.permission = true
      this.state.enabled = true
      this.trigger(this.state)
    }
  },

  chatUpdate: function(state) {
    var lastMsg = _.last(state.messages)
    if (!lastMsg) {
      return
    }

    if (!this._lastMsg) {
      this._lastMsg = lastMsg.time
      return
    }

    if (lastMsg.time > this._lastMsg) {
      this.notify('new message', {body: lastMsg.sender + ': ' + lastMsg.content})
    }
  },

  closeNotification: function() {
    if (this.notification) {
      this.notification.close()
    }
  },

  notify: function(message, options) {
    if (this.focus && !this.state.enabled || this.notification) {
      return
    }

    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout)
    }

    this.notification = new Notification(message, options)
    this.notification.onclick = function() {
      window.focus()
    }
  },
})
