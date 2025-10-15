document.addEventListener("DOMContentLoaded", function () {
  let el = document.getElementById("page-data")

  console.log("el.dataset payments: ", el.dataset.payments)
  let merchantId = el.dataset.merchantId
  let pageUid = el.dataset.pageUid
  let storeName = el.dataset.storeName || "Store"
  // let amountCents = parseInt(el.dataset.amountCents || "0", 10) // todo have this be the balance instead
  let amountCents = 1 //todo temp
  let currency = el.dataset.currency || "USD"
  let applePayMid = el.dataset.applePayMid || ""
  let googlePayMid = el.dataset.googlePayMid || ""
  let webTokenMid = el.dataset.webtokenMid || ""

  let methodCredit = document.getElementById("method-credit")
  let methodApple = document.getElementById("method-apple")
  let methodAppleWrapper = document.getElementById("method-apple-wrapper")
  let manualSection = document.getElementById("manual-section")
  let applePaySection = document.getElementById("apple-pay-section")

  // Desktop elements
  let methodCreditDesktop = document.getElementById("method-credit-desktop")
  let methodAppleDesktop = document.getElementById("method-apple-desktop")
  let methodAppleWrapperDesktop = document.getElementById("method-apple-wrapper-desktop")
  let manualSectionDesktop = document.getElementById("manual-section-desktop")
  let applePaySectionDesktop = document.getElementById("apple-pay-section-desktop")


  let chargeUrl = "/api/payments/" + merchantId + "/" + pageUid + "/charge"

  try {
    const rawPercentages = JSON.parse(el.dataset.allowedTipPercentages || "[]")
    // Convert from whole numbers (15, 18, 20) to decimals (0.15, 0.18, 0.20)
    allowedTipPercentages = rawPercentages.map((p) => p / 100)
  } catch (e) {
    console.error("Error parsing tip percentages:", e)
    allowedTipPercentages = [0.15, 0.18, 0.2] // Default fallback
  }


  // Function to format amount (helper function)
  function formatAmount(cents, currency) {
    const amount = (cents / 100).toFixed(2)
    return currency === "USD" ? `$${amount}` : `${amount} ${currency}`
  }

  function updateAmountDisplay() {
    const subtotalElement = document.getElementById("subtotal-display")
    const subtotalElementDesktop = document.getElementById("subtotal-display-desktop")
    const amountDisplayElement = document.getElementById("amount-display")
    const amountDisplayElementDesktop = document.getElementById("amount-display-desktop")

    // Calculate subtotal: AmountCents - tax - surcharge
    let totalFeesCents = 0

    // Add tax if present
    const taxElement = document.getElementById("tax-display")
    if (taxElement && taxElement.textContent) {
      const taxText = taxElement.textContent.replace(currency, "").replace("$", "").replace(/,/g, "").trim()
      const taxNumber = parseFloat(taxText)
      const taxCents = Math.round((isNaN(taxNumber) ? 0 : taxNumber) * 100)
      if (!isNaN(taxCents)) {
        totalFeesCents += taxCents
      }
    }

    // Add surcharge if present
    const surchargeElement = document.getElementById("surcharge-display")
    if (surchargeElement && surchargeElement.textContent) {
      const surchargeText = surchargeElement.textContent.replace(currency, "").replace("$", "").replace(/,/g, "").trim()
      const surchargeNumber = parseFloat(surchargeText)
      const surchargeCents = Math.round((isNaN(surchargeNumber) ? 0 : surchargeNumber) * 100)
      if (!isNaN(surchargeCents)) {
        totalFeesCents += surchargeCents
      }
    }

    // Calculate subtotal: AmountCents - tax - surcharge
    const subtotalCents = Math.max(0, amountCents - totalFeesCents)

    // Calculate amount due: AmountCents + tip
    const amountDueCents = amountCents

    // Update subtotal display
    if (subtotalElement) {
      subtotalElement.textContent = formatAmount(subtotalCents, currency)
    }
    if (subtotalElementDesktop) {
      subtotalElementDesktop.textContent = formatAmount(subtotalCents, currency)
    }

    // Update amount due display
    if (amountDisplayElement) {
      amountDisplayElement.textContent = formatAmount(amountDueCents, currency)
    }
    if (amountDisplayElementDesktop) {
      amountDisplayElementDesktop.textContent = formatAmount(amountDueCents, currency)
    }

    // Update global total for Apple Pay
    totalAmountCents = amountDueCents
  }


  // Theme Management
  function initializeTheme() {
    const themeToggle = document.getElementById("theme-toggle")
    const themeIcon = document.getElementById("theme-icon")
    const html = document.documentElement

    // Check for saved theme preference or default to system preference
    const savedTheme = localStorage.getItem("theme")
    const systemPrefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches

    let currentTheme = savedTheme || (systemPrefersDark ? "dark" : "light")

    // Apply theme
    function applyTheme(theme) {
      html.setAttribute("data-theme", theme)
      localStorage.setItem("theme", theme)

      // Update icon
      if (theme === "dark") {
        themeIcon.innerHTML =
          '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />'
      } else {
        themeIcon.innerHTML =
          '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />'
      }
    }

    // Toggle theme
    function toggleTheme() {
      currentTheme = currentTheme === "light" ? "dark" : "light"
      applyTheme(currentTheme)
    }

    // Listen for system theme changes
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (e) => {
      if (!localStorage.getItem("theme")) {
        currentTheme = e.matches ? "dark" : "light"
        applyTheme(currentTheme)
      }
    })

    // Apply initial theme
    applyTheme(currentTheme)

    // Add click listener
    themeToggle.addEventListener("click", toggleTheme)
  }

  initializeTheme()

  // Initialize date display
  try {
    let dateEl = document.getElementById("receipt-date")
    let dateElDesktop = document.getElementById("receipt-date-desktop")
    let createdAtUtc = el.dataset.createdAtUtc
    if (createdAtUtc) {
      let d = new Date(createdAtUtc)
      if (!isNaN(d.getTime())) {
        const dateString = d.toLocaleString()
        if (dateEl) dateEl.textContent = dateString
        if (dateElDesktop) dateElDesktop.textContent = dateString
      }
    }
  } catch (_) {}

  // Initialize items display with local data (will be updated by API call)
  try {
    let itemsRoot = document.getElementById("items_list")
    let itemsRootDesktop = document.getElementById("items_list_desktop")
    let itemsJSON = el.dataset.itemsJson
    if (itemsJSON) {
      let arr = JSON.parse(itemsJSON)
      if (Array.isArray(arr)) {
        const itemsHTML = arr
          .map(function (it) {
            let title = (it && it.title) || ""
            let desc = (it && it.description) || ""
            let price = (it && it.price) || 0
            let quantity = (it && it.quantity) || 0
            let total = (it && it.total) || 0

            return `<div class="flex items-center justify-between">
                             <div>
                               <div class="font-medium">${title}</div>
                               <div class="text-[12px] text-slate-500">${desc}</div>
                             </div>
                             <div class="font-mono text-sm">${(Number(price) / 100).toFixed(2)}</div>
                           </div>`
          })
          .join("")

        if (itemsRoot) itemsRoot.innerHTML = itemsHTML
        if (itemsRootDesktop) itemsRootDesktop.innerHTML = itemsHTML
      }
    }
  } catch (_) {}


  let msg = document.getElementById("msg")
  function setMsg(text) {
    msg.textContent = text || ""
    if (text && /^Error:/i.test(text)) {
      msg.classList.remove("text-slate-500")
      msg.classList.add("font-semibold", "text-red-700")
    } else {
      msg.classList.remove("font-semibold", "text-red-700")
      msg.classList.add("text-slate-500")
    }
    if (text && typeof text === "string" && /^Approved:/i.test(text)) {
      try {
        window.location.reload()
      } catch (_) {}
    }
  }

  function handleChargeWithToken(datacapToken, last4, brand) {
    return fetch(chargeUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        datacap_token: datacapToken,
        last4: last4,
        brand: brand,
        tip_amount_cents: 0, // todo tips later
        amount_cents: amountCents,
      }),
    }).then(async function (res) {
      let body = {}
      try {
        body = await res.json()
      } catch (e) {}
      if (!res.ok) throw new Error((body && body.message) || "Charge failed")
      if (body && body.approved) {
        return "Approved: " + (body.message || "")
      }
      throw new Error("Declined: " + ((body && body.message) || ""))
    })
  }

  // manual card
  let payBtn = document.getElementById("pay_button")
  let payBtnSpinner = document.getElementById("pay_btn_spinner")
  let payBtnText = document.getElementById("pay_btn_text")
  let paymentErrors = document.getElementById("payment_errors")

  // desktop elements
  let payBtnDesktop = document.getElementById("pay_button_desktop")
  let payBtnSpinnerDesktop = document.getElementById("pay_btn_spinner_desktop")
  let payBtnTextDesktop = document.getElementById("pay_btn_text_desktop")
  let paymentErrorsDesktop = document.getElementById("payment_errors_desktop")
  function clearErrors() {
    if (paymentErrors) paymentErrors.textContent = ""
    if (paymentErrorsDesktop) paymentErrorsDesktop.textContent = ""
  }
  function setPayLoading(isLoading) {
    try {
      // Mobile elements
      if (payBtn) {
        payBtn.disabled = !!isLoading
        payBtn.setAttribute("aria-busy", isLoading ? "true" : "false")
      }
      if (payBtnSpinner) payBtnSpinner.classList.toggle("hidden", !isLoading)
      if (payBtnText) payBtnText.textContent = isLoading ? "Processing..." : "Pay"

      // Desktop elements
      if (payBtnDesktop) {
        payBtnDesktop.disabled = !!isLoading
        payBtnDesktop.setAttribute("aria-busy", isLoading ? "true" : "false")
      }
      if (payBtnSpinnerDesktop) payBtnSpinnerDesktop.classList.toggle("hidden", !isLoading)
      if (payBtnTextDesktop) payBtnTextDesktop.textContent = isLoading ? "Processing..." : "Pay"
    } catch (_) {}
  }
  const tokenCallback = function (response) {
    setPayLoading(false)
    setMsg("")

    if (!response) {
      if (paymentErrors) paymentErrors.textContent = "No response from tokenization"
      if (paymentErrorsDesktop) paymentErrorsDesktop.textContent = "No response from tokenization"
      return
    }
    if (response.Error) {
      if (paymentErrors) paymentErrors.textContent = response.Error
      if (paymentErrorsDesktop) paymentErrorsDesktop.textContent = response.Error
      return
    }

    const token = response.Token || response.token || response.id
    if (!token) {
      if (paymentErrors) paymentErrors.textContent = "No token returned"
      if (paymentErrorsDesktop) paymentErrorsDesktop.textContent = "No token returned"
      return
    }

    let last4
    let brand
    try {
      last4 = response.Last4 || response.last4 || response.cardLast4 || undefined
      brand = response.Brand || response.brand || response.CardBrand || undefined
      console.log("[Datacap] Tokenization response:", response)
    } catch (_) {}
    handleChargeWithToken(token, last4, brand)
      .then(function (ok) {
        setMsg(ok)
      })
      .catch(function (err) {
        setMsg("Error: " + (err && err.message ? err.message : String(err)))
      })
      .finally(function () {
        setPayLoading(false)
      })
  }
  // Mobile pay button
  if (payBtn) {
    payBtn.addEventListener("click", function () {
      if (payBtn.disabled || payBtn.getAttribute("aria-busy") === "true") return
      clearErrors()
      setMsg("")
      setPayLoading(true)
      if (!window.DatacapWebToken) {
        if (paymentErrors) paymentErrors.textContent = "Token library unavailable"
        setPayLoading(false)
        return
      }
      console.log("requesting token")
      DatacapWebToken.requestToken(webTokenMid, "payment_form", tokenCallback)
    })
  }

  // Desktop pay button
  if (payBtnDesktop) {
    payBtnDesktop.addEventListener("click", function () {
      if (payBtnDesktop.disabled || payBtnDesktop.getAttribute("aria-busy") === "true") return
      clearErrors()
      setMsg("")
      setPayLoading(true)
      if (!window.DatacapWebToken) {
        if (paymentErrorsDesktop) paymentErrorsDesktop.textContent = "Token library unavailable"
        setPayLoading(false)
        return
      }
      console.log("requesting token")
      DatacapWebToken.requestToken(webTokenMid, "payment_form_desktop", tokenCallback)
    })
  }

  // Function to toggle between payment methods
  function togglePaymentMethod() {
    // Mobile
    if (methodCredit && methodCredit.checked) {
      if (manualSection) manualSection.style.display = "block"
      if (applePaySection) applePaySection.style.display = "none"
    } else if (methodApple && methodApple.checked) {
      if (manualSection) manualSection.style.display = "none"
      if (applePaySection) applePaySection.style.display = "block"
    }

    // Desktop
    if (methodCreditDesktop && methodCreditDesktop.checked) {
      if (manualSectionDesktop) manualSectionDesktop.style.display = "block"
      if (applePaySectionDesktop) applePaySectionDesktop.style.display = "none"
    } else if (methodAppleDesktop && methodAppleDesktop.checked) {
      if (manualSectionDesktop) manualSectionDesktop.style.display = "none"
      if (applePaySectionDesktop) applePaySectionDesktop.style.display = "block"
    }
  }

  // Add event listeners to radio buttons
  if (methodCredit) methodCredit.addEventListener("change", togglePaymentMethod)
  if (methodApple) methodApple.addEventListener("change", togglePaymentMethod)
  if (methodCreditDesktop) methodCreditDesktop.addEventListener("change", togglePaymentMethod)
  if (methodAppleDesktop) methodAppleDesktop.addEventListener("change", togglePaymentMethod)

  const totalAmount = amountCents
  if (window.DatacapApplePay && DatacapApplePay.init) {
    console.log("Initializing Apple Pay ... ", webTokenMid, storeName, applePayMid, (totalAmount / 100).toFixed(2))
    DatacapApplePay.init(tokenCallback, webTokenMid, storeName, applePayMid, (totalAmount / 100).toFixed(2))
  }

  // Set initial state (default to credit card)
  updateAmountDisplay()

  togglePaymentMethod()


  // dont have google pay mid yet
  // DatacapGooglePay.init(tokenCallback, webTokenMid ,merchantId,googlePayMid,(amountCents / 100).toFixed(2))
})
