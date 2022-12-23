if (
    // only forward amounts larger than 100 sat
    HtlcForward.Event.OutgoingAmountMsat >= 10000
) { true } else { false }

