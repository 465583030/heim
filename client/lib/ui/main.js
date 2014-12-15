var React = require('react/addons')
var Reflux = require('reflux')
var cx = React.addons.classSet

var actions = require('../actions')
var Scroller = require('./scroller')
var Chat = require('./chat')
var NotifyToggle = require('./notifytoggle')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat').store, 'chat'),
  ],

  getInitialState: function() {
    return {formFocus: false, settingsOpen: false}
  },

  send: function(ev) {
    if (ev.which != '13') {
      return
    }

    var input = this.refs.input.getDOMNode()
    actions.sendMessage(input.value)
    input.value = ''
    ev.preventDefault()
  },

  setNick: function(ev) {
    var input = this.refs.nick.getDOMNode()
    actions.setNick(input.value)
    ev.preventDefault()
  },

  previewNick: function() {
    var input = this.refs.nick.getDOMNode()
    this.setState({nickText: input.value})
  },

  focusInput: function(ev) {
    if (ev.target.nodeName == 'INPUT' || window.getSelection().type == 'Range') {
      return
    }

    var input = this.refs.input || this.refs.nick
    input.getDOMNode().focus()
  },

  onFormFocus: function() {
    this.setState({formFocus: true})
  },

  onFormBlur: function() {
    this.setState({formFocus: false})
  },

  toggleSettings: function() {
    this.setState({settingsOpen: !this.state.settingsOpen})
  },

  render: function() {
    var sendForm
    if (this.state.chat.nick) {
      sendForm = (
        <form className={cx({'focus': this.state.formFocus})}>
          <div className="nick-box">
            <div className="auto-size-container">
              <input className="nick" ref="nick" defaultValue={this.state.chat.nick} onBlur={this.setNick} onChange={this.previewNick} />
              <span className="nick">{this.state.nickText || this.state.chat.nick}</span>
            </div>
          </div>
          <input key="msg" ref="input" type="text" autoFocus disabled={this.state.chat.connected == false} onKeyDown={this.send} onFocus={this.onFormFocus} onBlur={this.onFormBlur} />
        </form>
      )
    } else {
      sendForm = (
        <form onSubmit={this.setNick} className={cx({'focus': this.state.formFocus})}>
          <label>choose a nickname to start chatting:</label>
          <input key="nick" ref="nick" type="text" onFocus={this.onFormFocus} onBlur={this.onFormBlur} />
        </form>
      )
    }

    return (
      <div className="chat">
        <Scroller className={cx({'messages-container': true, 'settings-open': this.state.settingsOpen})} onClick={this.focusInput}>
          <div className="messages-content">
            {sendForm}
            <button type="button" className="settings" onClick={this.toggleSettings} />
            <Chat messages={this.state.chat.messages} hues={this.state.chat.nickHues} disconnected={this.state.chat.connected == false} />
          </div>
          <div className="settings-pane">
            <NotifyToggle />
          </div>
        </Scroller>
      </div>
    )
  },
})
