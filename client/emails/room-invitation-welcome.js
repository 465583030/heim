import React from 'react'

import { Item, Text, Link } from './email'
import { StandardEmail, TopBubbleBox, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo.png">
      <Item align="center">
        <Text {...textDefaults} fontSize={18}>hi! <strong>{'{{.SenderName}}'}</strong> invites you to join a {'{{.RoomPrivacy}}'} chat room:</Text>
      </Item>
      <Item align="center">
        <Link href="https://euphoria.io/room/space">
          <Text {...textDefaults} fontSize={32} color={null}>&{'{{.RoomName}}'}</Text>
        </Link>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Text {...textDefaults} color="#7d7d7d">a note from @{'{{.SenderName}}'}:</Text>
      </Item>
      <Item>
        <Text {...textDefaults}>{'{{.SenderMessage}}'}</Text>
      </Item>
    </BodyBox>
    <BodyBox>
      <Item>
        <Text {...textDefaults}><Link href="{{.RoomURL}}">&{'{{.RoomName}}'}</Link> is hosted on <Link {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</Link>, a free online discussion platform. you don't have to sign up to chat &ndash; just click the link, enter a nickname, and you'll be chatting with {'{{.SenderName}}'} in moments.</Text>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
