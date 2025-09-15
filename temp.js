function DatacapApplePay() {}
DatacapApplePay.isRecurringPayment = !1;
DatacapApplePay.recurringDescription = null;
DatacapApplePay.recurringLabel = null;
DatacapApplePay.managementURL = null;
DatacapApplePay.beginPayment = function (n) {
  var t;
  n.preventDefault();
  var r = DatacapApplePay.storeName,
    u = Number(DatacapApplePay.subTotal),
    e = { label: r + "", type: "final", amount: u },
    f = ["email", "name", "phone", "postalAddress"],
    i = {
      countryCode: "US",
      currencyCode: "USD",
      merchantCapabilities: ["supports3DS"],
      supportedNetworks: ["amex", "masterCard", "visa", "discover"],
      total: e,
      requiredBillingContactFields: f,
      requiredShippingContactFields: f,
    };
  DatacapApplePay.isRecurringPayment &&
    ((i.recurringPaymentRequest = {
      paymentDescription:
        DatacapApplePay.recurringDescription || "Subscription",
      regularBilling: {
        label: DatacapApplePay.recurringLabel || r + " Subscription",
        amount: u,
        type: "final",
        paymentTiming: "recurring",
        recurringPaymentStartDate: new Date().toISOString(),
        recurringPaymentIntervalUnit:
          DatacapApplePay.recurringIntervalUnit || "month",
        recurringPaymentIntervalCount:
          DatacapApplePay.recurringIntervalCount || 1,
      },
      managementURL: DatacapApplePay.managementURL || window.location.origin,
    }),
    console.log(
      "Apple Pay recurring payment enabled:",
      i.recurringPaymentRequest
    ));
  t = new ApplePaySession(6, i);
  t.onvalidatemerchant = function (n) {
    var r = {
        validationUrl: n.validationURL,
        merchantStoreName: DatacapApplePay.storeName,
        appleMerchantID: DatacapApplePay.appleMerchantID,
        merchantHostName: window.location.hostname,
      },
      i = {};
    i[DatacapApplePay.xAntiforgeryName] = DatacapApplePay.xAntiforgeryToken;
    i["Content-Type"] = "application/json; charset=utf-8";
    fetch("https://wallet.dcap.com/applepay/validate", {
      method: "POST",
      body: JSON.stringify(r),
      headers: i,
    })
      .then(function (n) {
        if (!n.ok) {
          var i = {
            Error:
              "Failed to validate merchant. Contact Datacap support. " +
              n.statusText,
          };
          DatacapApplePay.callback(i);
          return;
        }
        n.json().then(function (n) {
          if (n.statusCode) {
            var i = {
              Error:
                "Failed to validate merchant. Contact Datacap support. Message: " +
                n.statusMessage,
            };
            DatacapApplePay.callback(i);
            return;
          }
          t.completeMerchantValidation(n);
        });
      })
      .catch(function (n) {
        var t = {
          Error:
            "Failed to validate merchant. Contact Datacap support. Message: " +
            n.message,
        };
        DatacapApplePay.callback(t);
        return;
      });
  };
  t.onpaymentauthorized = function (n) {
    var r = {},
      i;
    r[DatacapApplePay.xAntiforgeryName] = DatacapApplePay.xAntiforgeryToken;
    r["Content-Type"] = "application/json; charset=utf-8";
    i = {};
    i.token = n.payment.token;
    i.tokenKey = DatacapApplePay.tokenKey;
    fetch("https://wallet.dcap.com/applepay/tokenize", {
      method: "POST",
      body: JSON.stringify(i),
      headers: r,
    }).then(function (i) {
      var r, u;
      if (i.status !== 200) {
        r = { Error: "Failed to create token" };
        DatacapApplePay.callback(r);
        u = { status: 1 };
        t.completePayment(u);
        return;
      }
      i.json().then(function (i) {
        async function r() {
          var n;
          try {
            const r = await DatacapApplePay.callback(i);
            n = { status: 0 };
            t.completePayment(n);
          } catch (r) {
            n = { status: 1 };
            t.completePayment(n);
          }
        }
        n.payment.shippingContact &&
          (i.Customer = {
            FirstName: n.payment.shippingContact.givenName,
            LastName: n.payment.shippingContact.familyName,
            Address: n.payment.shippingContact.addressLines,
            City: n.payment.shippingContact.locality,
            State: n.payment.shippingContact.administrativeArea,
            Zip: n.payment.shippingContact.postalCode,
            Email: n.payment.shippingContact.emailAddress,
            Phone: n.payment.shippingContact.phoneNumber,
          });
        r();
      });
    });
  };
  t.begin();
};
DatacapApplePay.showButton = function () {
  fetch(
    "https://wallet.dcap.com/applepay/supports3ds/" + DatacapApplePay.tokenKey
  ).then(function (n) {
    n.ok &&
      n.text().then(function (n) {
        if (n == "true") {
          var t = document.getElementById("apple-pay-button");
          t.setAttribute("lang", DatacapApplePay.getPageLanguage());
          t.setAttribute("onclick", "DatacapApplePay.beginPayment(event)");
          t.classList.add("apple-pay");
          t.classList.add("input-block-level");
          t.classList.add("apple-pay-button");
          t.classList.add("apple-pay-button-black");
        }
      });
  });
};
DatacapApplePay.supportedByDevice = function () {
  return "ApplePaySession" in window;
};
DatacapApplePay.getPageLanguage = function () {
  return document.documentElement.lang || "en";
};
DatacapApplePay.loadApplePayCSS = function () {
  var n = document.createElement("link");
  n.rel = "stylesheet";
  n.href = "https://wallet.dcap.com/css/applepay.css";
  document.getElementsByTagName("head")[0].appendChild(n);
};
DatacapApplePay.init = function (n, t, i, r, u, f) {
  var o, e;
  if (
    ((DatacapApplePay.callback = n),
    (DatacapApplePay.tokenKey = t),
    (DatacapApplePay.storeName = i),
    (DatacapApplePay.appleMerchantID = r),
    (DatacapApplePay.subTotal = u),
    (DatacapApplePay.xAntiforgeryName = "x-antiforgery-token"),
    (DatacapApplePay.xAntiforgeryToken =
      "CfDJ8Naziwl2XYVOlPePzKp8Fu_Sdd3rf5953rtJn7zSao8_xZxUilKvo2YAVDWdk-u5vk9IMcbnrBp_j5uIaQDndj_vfrsIt56UJggE85nLC6M7yRi50x2vaQ0V6vLjqRzkr3O7jC4aBRQvoLDvvOpNmNo"),
    f && f.isRecurring
      ? ((DatacapApplePay.isRecurringPayment = !0),
        (DatacapApplePay.recurringDescription =
          f.description || "Subscription"),
        (DatacapApplePay.recurringLabel = f.label || i + " Subscription"),
        (DatacapApplePay.managementURL =
          f.managementURL || window.location.origin),
        (DatacapApplePay.recurringIntervalUnit = f.intervalUnit || "month"),
        (DatacapApplePay.recurringIntervalCount = f.intervalCount || 1))
      : ((DatacapApplePay.isRecurringPayment = !1),
        (DatacapApplePay.recurringDescription = null),
        (DatacapApplePay.recurringLabel = null),
        (DatacapApplePay.managementURL = null),
        (DatacapApplePay.recurringIntervalUnit = null),
        (DatacapApplePay.recurringIntervalCount = null)),
    DatacapApplePay.appleMerchantID)
  ) {
    if (DatacapApplePay.supportedByDevice()) {
      if (((o = document.getElementById("apple-pay-button")), !o)) {
        e = { Error: "Unable to find Apple Pay div with ID apple-pay-button." };
        n(e);
        return;
      }
      DatacapApplePay.loadApplePayCSS();
      ApplePaySession.canMakePayments() === !0
        ? DatacapApplePay.showButton()
        : ApplePaySession.canMakePaymentsWithActiveCard(
            DatacapApplePay.appleMerchantID
          ).then(function (t) {
            if (t === !0) DatacapApplePay.showButton();
            else
              n({
                Error:
                  "Browser or device does not support Apple Pay on the web.",
              });
          });
    }
  } else
    (e = { Error: "No Apple Pay merchant identifier is configured." }), n(e);
};
DatacapApplePay.updateAmountValue = function (n) {
  DatacapApplePay.subTotal = n;
};
