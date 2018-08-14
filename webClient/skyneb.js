'use strict'
var Skyneb = {
    token: function(){
      return token
    },

    connect: function (url,connect) {
        var xmlhttp = new XMLHttpRequest();
        xmlhttp.open("POST",url+"subscribe", "/json-handler");
        xmlhttp.onload = function () {
            var json_data = JSON.parse(xmlhttp.responseText);
            if (xmlhttp.readyState == 4 && xmlhttp.status == "200") {
                if( json_data.status_code == 330){
                    return json_data.status_message;
                } else if (json_data.status_code == 331){
                    return json_data.status_message;

                }else if(json_data.status_code == 320) {
                    return json_data.status_message;
                } else if (json_data.status_code == 321) {
                    return json_data.status_message;
                } else {
                    return json_data;
                }
            } else {
                return json_data;

            }
        };
        var datas = JSON.stringify(connect);
        xmlhttp.send(datas);

    },

    publish: function (url,send_data) {
        var xmlhttp = new XMLHttpRequest();
        xmlhttp.open("POST",url+"publish", "/json-handler");
        xmlhttp.onload = function () {
            var json_data = JSON.parse(xmlhttp.responseText);
            if (xmlhttp.readyState == 4 && xmlhttp.status == "200") {
                if( json_data.status_code == 200){
                    return json_data.status_message;
                } else if (json_data.status_code == 354){
                    return json_data.status_message;

                }else if(json_data.status_code == 353) {
                    return json_data.status_message;
                } else if (json_data.status_code == 352) {
                    alert(json_data);
                    return json_data.status_message;
                } else if (json_data.status_code == 351){
                    return json_data.status_message;

                }else {
                    return json_data;
                }
            } else {
                return json_data;

                console.log(json_data);
            }
        };

        var send_data = JSON.stringify(send_data);

        xmlhttp.send(send_data);
    }



}
