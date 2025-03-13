export default {
    async fetch(request, env) {
        if (request.headers.get("authorization") !== 'MY_AUTH_KEY') {
            return new Response("Sorry, you have supplied an invalid key.", {
                status: 403,
            });
        }

        try {
            const file = await request.text();
            const model = "@cf/llava-hf/llava-1.5-7b-hf"

            var prompt = request.headers.get("prompt")
            if (!prompt) {
                prompt = "Generate a detailed caption for this image"
            }

            const image = [...new Uint8Array(file.arrayBuffer)]

            var response = {}
            var now;

            now = new Date();
            response = await env.AI.run(model, {
                image: image,
                prompt: prompt,
                max_tokens: 512,
            });
            response.elapsedTimeMs = new Date() - now;
            response.prompt = prompt;
            response.model = model

            return Response.json(response);
        }
        catch(err) {
            return new Response(err.message, {
                status: 500,
            });
        }
    }
};

// time curl --data-binary '@IMG_5373.jpg' -H "Authorization: MY_AUTH_KEY" -H "Prompt: Write a short fairy tale based on the picture" https://llava.xxxxxx.workers.dev/
