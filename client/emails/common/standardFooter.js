import React from 'react'

import { Link, Text, textDefaults } from '../email'
import Footer from './Footer'


export default (
  <Footer>
    <Text {...textDefaults} fontSize={13} color="#7d7d7d">
      this message was sent to <Link {...textDefaults} textDecoration="none" href="mailto:{{.AccountEmailAddress}}">{'{{.AccountEmailAddress}}'}</Link> because an account is registered on <Link {...textDefaults} textDecoration="none" href="{{.SiteURL}}">{'{{.SiteURLShort}}'}</Link> with this email address.
      would you like to change your <Link {...textDefaults} href="{{.EmailPreferencesURL}}">email notification preferences</Link>?
    </Text>
  </Footer>
)
