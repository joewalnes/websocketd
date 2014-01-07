import java.io.*;
public class Echo {
    public static void main(String[] args) {
        while(true) {
            try {
                BufferedReader in = new BufferedReader(new InputStreamReader(System.in));
                String message = in.readLine();
                System.out.println(message);
            } catch (IOException e) {
                e.printStackTrace();
            }
        }
    }
}