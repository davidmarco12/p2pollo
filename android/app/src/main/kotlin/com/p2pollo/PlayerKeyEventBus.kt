package com.p2pollo

import android.view.KeyEvent
import kotlinx.coroutines.flow.MutableSharedFlow

/** Bus para propagar eventos de teclas desde la Activity hasta PlayerScreen. */
object PlayerKeyEventBus {
    val events = MutableSharedFlow<KeyEvent>(extraBufferCapacity = 16)
}
