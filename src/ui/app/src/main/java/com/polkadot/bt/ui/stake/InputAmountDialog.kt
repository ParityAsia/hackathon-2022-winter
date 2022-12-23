package com.polkadot.bt.ui.stake

import android.app.Dialog
import android.content.Context
import android.content.res.ColorStateList
import android.graphics.Color
import android.view.View
import android.widget.ImageView
import android.widget.TextView
import androidx.appcompat.widget.AppCompatEditText
import androidx.core.view.isVisible
import androidx.core.widget.addTextChangedListener
import com.polkadot.bt.R
import com.polkadot.bt.ext.clickNoRepeat
import com.polkadot.bt.ext.isNotEmpty
import com.polkadot.bt.ext.string
import com.polkadot.bt.ext.toast

class InputAmountDialog(context: Context, perform: (String) -> Unit) : Dialog(context, R.style.MBottomSheetDialog) {

    init {
        val view = View.inflate(context, R.layout.input_amount_dialog, null)
        val password: AppCompatEditText = view.findViewById(R.id.password)
        val determine: TextView = view.findViewById(R.id.determine)
        val cancel: TextView = view.findViewById(R.id.cancel)
        val delete: ImageView = view.findViewById(R.id.delete_content)
        determine.isEnabled = false
        password.addTextChangedListener {
            if (it.toString().length > 0) {
                determine.isEnabled = true
                determine.backgroundTintList = ColorStateList.valueOf(Color.parseColor("#FF1B1B1C"))
            } else {
                determine.isEnabled = false
                determine.backgroundTintList = ColorStateList.valueOf(Color.parseColor("#FFDCDCE0"))
            }
            delete.isVisible = it.toString().isNotEmpty()
        }

        delete.setOnClickListener {
            password.setText("")
        }
        determine.clickNoRepeat {
            if (password.isNotEmpty()) {
                dismiss()
                perform.invoke(password.string())
            } else {
                toast("password must 8 words")
            }
        }
        cancel.clickNoRepeat {
            dismiss()
        }
        setContentView(view)
        setCancelable(true)
        setCanceledOnTouchOutside(true)
    }
}