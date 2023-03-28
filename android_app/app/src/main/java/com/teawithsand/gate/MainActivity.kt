package com.teawithsand.gate

import android.app.PendingIntent
import android.content.Intent
import android.os.Bundle
import android.util.Log
import android.widget.Button
import androidx.appcompat.app.AppCompatActivity
import com.google.android.gms.location.Geofence
import com.google.android.gms.location.GeofencingRequest
import com.google.android.gms.location.LocationServices
import nl.altindag.ssl.SSLFactory
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.security.KeyStore
import java.util.concurrent.Executors
import javax.net.ssl.*


class MainActivity : AppCompatActivity() {
    val executor by lazy {
        Executors.newFixedThreadPool(1)
    }
    val geofencingClient by lazy {
        LocationServices.getGeofencingClient(this)
    }
    val client by lazy {
        val keyStore = KeyStore.getInstance("BKS")
        keyStore.load(getCertificateStream(), "asdfasdf".toCharArray())
        val verifier: HostnameVerifier = HostnameVerifier { hostname, session -> true }

        val fac = SSLFactory.builder()
            .withIdentityMaterial(getCertificateStream(), "asdfasdf".toCharArray())
            .withTrustMaterial(getCertificateStream(), "asdfasdf".toCharArray())
            .withHostnameVerifier(verifier)
            .build()


        val client = OkHttpClient().newBuilder()
            .hostnameVerifier(verifier)
            .sslSocketFactory(fac.sslSocketFactory, fac.trustManager.get() as X509TrustManager)
            .build()

        client
    }
    private val geofencePendingIntent: PendingIntent by lazy {
        val intent = Intent(this, GeofenceBroadcastReceiver::class.java)
        // We use FLAG_UPDATE_CURRENT so that we get the same pending intent back when calling
        // addGeofences() and removeGeofences().
        PendingIntent.getBroadcast(this, 0, intent, PendingIntent.FLAG_UPDATE_CURRENT)
    }
    private val list by lazy {
        geofenceList.add(Geofence.Builder()
            // Set the request ID of the geofence. This is a string to identify this
            // geofence.
            .setRequestId("geofence id ")

            // Set the circular region of this geofence.
            .setCircularRegion(
                entry.value.latitude,
                entry.value.longitude,
                Constants.GEOFENCE_RADIUS_IN_METERS
            )

            // Set the expiration duration of the geofence. This geofence gets automatically
            // removed after this period of time.
            .setExpirationDuration(Constants.GEOFENCE_EXPIRATION_IN_MILLISECONDS)

            // Set the transition types of interest. Alerts are only generated for these
            // transition. We track entry and exit transitions in this sample.
            .setTransitionTypes(Geofence.GEOFENCE_TRANSITION_ENTER or Geofence.GEOFENCE_TRANSITION_EXIT)

            // Create the geofence.
            .build())

    }


    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        geofencingClient.addGeofences(
            GeofencingRequest.Builder().apply {
                setInitialTrigger(GeofencingRequest.INITIAL_TRIGGER_ENTER)
                addGeofences(geofenceList)
            }.build(),
            geofencePendingIntent,
        )

        findViewById<Button>(R.id.lights).setOnClickListener {
            val req = Request.Builder()
                .url("https://gate.teawithsand.com:1997/light-button")
                .method("POST", ByteArray(0).toRequestBody())
                .build()

            executor.execute {
                try {
                    client.newCall(req).execute()
                } catch (e: Exception) {
                    Log.i("REQUESTER", "Request filed", e)
                }
            }
        }

        findViewById<Button>(R.id.open_gate).setOnClickListener {
            val req = Request.Builder()
                .url("https://gate.teawithsand.com:1997/open-gate")
                .method("POST", ByteArray(0).toRequestBody())
                .build()

            executor.execute {
                try {

                    client.newCall(req).execute()
                } catch (e: Exception) {
                    Log.i("REQUESTER", "Request filed", e)
                }
            }
        }
    }


}