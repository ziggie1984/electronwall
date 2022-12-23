// only channels > 2 Mio sats
ChannelAccept.Event.FundingAmt >= 2000000 &&
(
    // only nodes with Amboss contact data
    ChannelAccept.Amboss.Socials.Info.Email ||
    ChannelAccept.Amboss.Socials.Info.Twitter ||
    ChannelAccept.Amboss.Socials.Info.Telegram
) &&
// Only allow private channels which are smaller than 10 mio sats
(
        (ChannelAccept.Event.ChannelFlags & 1) == 0 &&
        ChannelAccept.Event.FundingAmt <= 10000000 ||
        // allow all public channels
        (ChannelAccept.Event.ChannelFlags & 1) == 1
);
