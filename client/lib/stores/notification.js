var fs = require('fs')
var _ = require('lodash')
var Reflux = require('reflux')
var Immutable = require('immutable')

var storage = require('./storage')
var chat = require('./chat')
var activity = require('./activity')


var favicons = module.exports.favicons = {
  'active': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-active.png', 'base64'),
  'highlight': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-highlight.png', 'base64'),
  'disconnected': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/favicon-disconnected.png', 'base64'),
}

var icons = module.exports.icons = {
  'active': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/icon.png', 'base64'),
  'highlight': 'data:image/png;base64,' + fs.readFileSync(__dirname + '/../../res/icon-highlight.png', 'base64'),
}

var storeActions = Reflux.createActions([
  'enablePopups',
  'disablePopups',
  'pausePopupsUntil',
  'setRoomNotificationMode',
  'dismissNotification',
])
_.extend(module.exports, storeActions)

storeActions.enablePopups.sync = true

module.exports.store = Reflux.createStore({
  listenables: [
    storeActions,
    {storageChange: storage.store},
    {chatStateChange: chat.store},
    {messageReceived: chat.messageReceived},
    {messagesChanged: chat.messagesChanged},
    {onActive: activity.becameActive},
    {onInactive: activity.becameInactive},
  ],

  timeout: 3 * 1000,

  seenExpirationTime: 30 * 1000,

  priority: {
    'new-message': 1,
    'new-reply': 2,
    'new-mention': 3,
  },

  init: function() {
    this.state = {
      popupsEnabled: null,
      popupsSupported: 'Notification' in window,
      popupsPausedUntil: null,
      soundEnabled: false,
      latestNotifications: Immutable.OrderedMap(),
      notifications: Immutable.OrderedMap(),
      newMessageCount: 0,
    }

    this.active = true
    this.joined = false
    this.connected = false
    this.alerts = {}
    this._roomStorage = null

    if (this.state.popupsSupported) {
      this.state.popupsPermission = Notification.permission == 'granted'
    }

    this._messages = null
    this._notified = {}
    this._dismissed = {}
    this._newNotifications = []
    this._queueUpdateNotifications = _.debounce(this._updateNotifications, 0)
    this._inactiveUpdateNotificationsTimeout = null

    window.addEventListener('unload', this.removeAllAlerts)
  },

  onActive: function() {
    clearTimeout(this._inactiveUpdateNotificationsTimeout)
    this.active = true
    this.removeAllAlerts()
    this._updateFavicon()
    this._updateTitleCount()
  },

  onInactive: function() {
    this.active = false
    this._inactiveUpdateNotificationsTimeout = setTimeout(this._updateNotifications, this.seenExpirationTime)
  },

  getInitialState: function() {
    return this.state
  },

  enablePopups: function() {
    if (this.state.popupsPermission) {
      storage.set('notify', true)
      storage.set('notifyPausedUntil', null)
    } else {
      Notification.requestPermission(this.onPermission)
    }
  },

  disablePopups: function() {
    storage.set('notify', false)
    storage.set('notifyPausedUntil', null)
  },

  pausePopupsUntil: function(time) {
    storage.set('notifyPausedUntil', Math.floor(time))
  },

  setRoomNotificationMode: function(roomName, mode) {
    storage.setRoom(roomName, 'notifyMode', mode)
  },

  onPermission: function(permission) {
    this.state.popupsPermission = permission == 'granted'
    if (this.state.popupsPermission) {
      storage.set('notify', true)
    }
    this.trigger(this.state)
  },

  storageChange: function(data) {
    if (!data) {
      return
    }

    var popupsWereEnabled = this._popupsAreEnabled()
    this.state.popupsEnabled = data.notify
    this.state.popupsPausedUntil = data.notifyPausedUntil
    if (popupsWereEnabled && !this._popupsAreEnabled()) {
      this.closeAllPopups()
    }

    this.state.soundEnabled = data.notifySound
    if (this.state.soundEnabled) {
      // preload audio file
      require('../alert-sound')
    }

    this._roomStorage = data.room
    this.trigger(this.state)
  },

  chatStateChange: function(chatState) {
    this.connected = chatState.connected
    this.joined = chatState.joined
    this._updateFavicon()
  },

  messageReceived: function(message, state) {
    if (message.get('_own')) {
      // dismiss notifications on siblings or parent of sent messages
      var parentId = message.get('parent')
      if (!parentId || parentId == '__root') {
        return
      }
      module.exports.dismissNotification(parentId)
      var parentMessage = state.messages.get(parentId)
      Immutable.Seq(parentMessage.get('children'))
        .forEach(id => module.exports.dismissNotification(id))
    }
  },

  messagesChanged: function(ids, state) {
    this._messages = state.messages
    var unseen = Immutable.Seq(ids)
      .map(id => {
        var msg = state.messages.get(id)

        if (id == '__root' || this.state.latestNotifications.has(id)) {
          this._queueUpdateNotifications()
        }

        // exclude already notified
        if (_.has(this._notified, id)) {
          return
        }

        if (!this._shouldShowNotification(msg, Date.now())) {
          return
        }

        if (msg.get('_own')) {
          return
        }

        return msg
      })
      .filter(Boolean)
      .cacheResult()

    unseen.forEach(msg => this._markNotification('new-message', state.roomName, msg))

    unseen
      .filter(msg => {
        var msgId = msg.get('id')
        var parentId = msg.get('parent')

        if (parentId == '__root') {
          return false
        }

        var parentMessage = state.messages.get(parentId)
        var children = parentMessage.get('children').toList()

        if (parentMessage.get('_own') && children.first() == msgId) {
          return true
        }

        var prevChild = children.get(children.indexOf(msgId) - 1)
        return prevChild && state.messages.get(prevChild).get('_own')
      })
      .forEach(msg => this._markNotification('new-reply', state.roomName, msg))

    unseen
      .filter(msg => msg.get('_mention'))
      .forEach(msg => this._markNotification('new-mention', state.roomName, msg))
  },

  _markNotification: function(kind, roomName, message) {
    this._newNotifications.push({
      kind: kind,
      roomName: roomName,
      message: message,
    })
    this._queueUpdateNotifications()
  },

  _shouldShowNotification: function(msg, now) {
    if (!msg) {
      return false
    }

    if (_.has(this._dismissed, msg.get('id'))) {
      return false
    }

    if (msg.get('deleted') || !msg.has('$count')) {
      return false
    }

    var seen = msg.get('_seen')
    if (seen === true || seen && seen <= now - this.seenExpirationTime) {
      return false
    }

    return true
  },

  _updateNotifications: function() {
    var now = Date.now()
    var alerts = {}

    var groups = this.state.latestNotifications
      .withMutations(notifications => {
        _.each(this._newNotifications, newNotification => {
          var newMessageId = newNotification.message.get('id')
          var existingNotification = notifications.get(newMessageId)
          var newPriority = this.priority[newNotification.kind]
          var oldPriority = existingNotification && this.priority[existingNotification.kind] || 0
          if (newPriority > oldPriority) {
            notifications.set(newMessageId, newNotification)
            alerts[newNotification.kind] = newNotification
            if (!this.active && !this._notified[newMessageId]) {
              this.state.newMessageCount++
            }
            this._notified[newMessageId] = true
          }
        })
      })
      .sortBy(notification => notification.message.get('time'))
      .groupBy(notification => notification.kind)

    var newMention = alerts['new-mention']
    if (newMention) {
      var newMentionId = newMention.message.get('id')
      alerts = _.reject(alerts, a => a.message.get('id') == newMentionId)
      this._notifyAlert('new-mention', newMention.roomName, newMention.message, {
        favicon: favicons.highlight,
        icon: icons.highlight,
      })
    }

    var newMessage = alerts['new-reply'] || alerts['new-message']
    if (newMessage) {
      this._notifyAlert(alerts['new-reply'] ? 'new-reply' : 'new-message', newMessage.roomName, newMessage.message, {
        favicon: favicons.active,
        icon: icons.active,
        timeout: this.timeout,
      })
    }

    this._updateFavicon()
    this._updateTitleCount()

    var empty = Immutable.OrderedMap()

    this.state.latestNotifications = empty.concat(
      groups.get('new-mention', empty),
      groups.get('new-reply', empty).takeLast(6),
      groups.get('new-message', empty).takeLast(3)
    )

    this.state.notifications = this.state.latestNotifications
      .filterNot((notification, id) => {
        if (!this._shouldShowNotification(this._messages.get(id), now)) {
          if (this.state.notifications.has(id)) {
            this.removeAlert(notification.kind, id)
          }
          return true
        }
      })
      .map(notification => notification.kind)

    this._newNotifications = []

    this.trigger(this.state)
  },

  _updateFavicon: function() {
    if (!this.connected) {
      Heim.setFavicon(favicons.disconnected)
    } else {
      var alert = this.alerts['new-mention'] || this.alerts['new-message']
      Heim.setFavicon(alert ? alert.favicon : '/static/favicon.png')
    }
  },

  _updateTitleCount: function() {
    Heim.setTitleMsg(this.state.newMessageCount || '')
  },

  dismissNotification: function(messageId) {
    var kind = this.state.notifications.get(messageId)
    if (kind) {
      this.removeAlert(kind, messageId)
      this._dismissed[messageId] = true
      this._queueUpdateNotifications()
      this.trigger(this.state)
    }
  },

  closePopup: function(kind) {
    var alert = this.alerts[kind]
    if (!alert) {
      return
    }
    clearTimeout(this.alerts[kind].timeout)
    // when we close a notification, its onclose callback will get called
    // async. displaying a new notification can race with this, causing the
    // new notification to be invalidly forgotten.
    if (alert.popup) {
      alert.popup.onclose = () => {
        alert.popup = null
      }
      alert.popup.close()

      // hack: sometimes chrome doesn't close notififcations when we tell it to
      // (while still animating showing it, perhaps?). this failsafe seems to
      // do the trick.
      setTimeout(() => {
        if (alert.popup !== null) {
          alert.popup.close()
        }
      }, 500)
    }
  },

  closeAllPopups: function() {
    _.each(this.alerts, (alert, kind) => this.closePopup(kind))
  },

  removeAlert: function(kind, messageId) {
    var alert = this.alerts[kind]
    if (!alert) {
      return
    }
    if (messageId && this.alerts[kind].messageId != messageId) {
      return
    }
    this.closePopup(kind)
    delete this.alerts[kind]
  },

  removeAllAlerts: function() {
    _.each(this.alerts, (alert, kind) => this.removeAlert(kind))
    this.state.newMessageCount = 0
  },

  _popupsAreEnabled: function() {
    if (!this.state.popupsEnabled) {
      return false
    }

    var popupsPaused = this.state.popupsPausedUntil && Date.now() < this.state.popupsPausedUntil
    return !popupsPaused
  },

  _shouldPopup: function(kind, roomName) {
    var priority = this.priority[kind]
    var notifyMode = _.get(this._roomStorage, [roomName, 'notifyMode'], 'mention')
    if (notifyMode == 'none') {
      return false
    } else if (priority < this.priority['new-' + notifyMode]) {
      return false
    }
    return true
  },

  _notifyAlert: function(kind, roomName, message, options) {
    if (this.active) {
      return
    }

    var shouldPopup = this._shouldPopup(kind, roomName)

    // note: alert state encompasses favicon state too, so we need to replace
    // the alert regardless of whether we're configured to show a popup

    if (kind == 'new-reply') {
      // have new reply notifications replace new messages and vice versa
      kind = 'new-message'
    }

    this.removeAlert(kind)

    var messageId = message.get('id')
    var alert = this.alerts[kind] = {}
    alert.messageId = messageId
    alert.favicon = options.favicon
    delete options.favicon

    if (!this.joined || !shouldPopup) {
      return
    }

    if (this.state.popupsPermission && this._popupsAreEnabled()) {
      if (this.state.soundEnabled && kind == 'new-mention') {
        require('../alert-sound').play()
      }

      var timeoutDuration = options.timeout
      delete options.timeout

      options.body = message.getIn(['sender', 'name']) + ': ' + message.get('content')

      alert.popup = new Notification(roomName, options)

      var ui = require('./ui')  // avoid import loop
      alert.popup.onclick = () => {
        uiwindow.focus()
        this.dismissNotification(messageId)
        ui.gotoMessageInPane(messageId)
      }
      alert.popup.onclose = _.partial(this.closePopup, kind)

      if (timeoutDuration) {
        alert.timeout = setTimeout(alert.popup.onclose, timeoutDuration)
      }
    }
  },
})
